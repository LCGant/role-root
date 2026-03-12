package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type result struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseTimeout() time.Duration {
	v := getenv("SMOKE_TIMEOUT", "60s")
	d, err := time.ParseDuration(v)
	if err != nil {
		return 60 * time.Second
	}
	return d
}

func main() {
	base := strings.TrimRight(getenv("GATEWAY_BASE", "http://gateway:8080"), "/")
	timeout := parseTimeout()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 10 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	scenarios := []func(context.Context, *http.Client, string) result{
		healthChecks,
		pdpAdminBlock,
		payloadLimit,
		authRegisterLoginLogout,
		authMFAFlow,
		authCSRFExtra,
		rateLimitGateway,
	}

	results := make([]result, 0, len(scenarios))
	for _, fn := range scenarios {
		res := fn(ctx, client, base)
		fmt.Printf("[%s] %s\n", res.Status, res.Name)
		results = append(results, res)
	}

	out, _ := json.MarshalIndent(results, "", "  ")
	fmt.Println(string(out))

	for _, r := range results {
		if r.Status != "ok" {
			os.Exit(1)
		}
	}
}

// ---- Scenarios ----

func healthChecks(ctx context.Context, c *http.Client, base string) result {
	paths := []string{"/healthz", "/auth/healthz", "/pdp/healthz"}
	for _, p := range paths {
		if err := expectStatus(ctx, c, base+p, http.StatusOK, nil, nil); err != nil {
			return fail("health_checks", err)
		}
	}
	return ok("health_checks")
}

func pdpAdminBlock(ctx context.Context, c *http.Client, base string) result {
	headers := map[string]string{"Accept": "application/json"}
	if err := expectStatus(ctx, c, base+"/pdp/v1/admin/status", http.StatusForbidden, nil, headers); err != nil {
		return fail("pdp_admin_block", err)
	}
	return ok("pdp_admin_block")
}

func payloadLimit(ctx context.Context, c *http.Client, base string) result {
	body := bytes.Repeat([]byte("A"), 1_500_000)
	if err := expectStatus(ctx, c, base+"/auth/healthz", http.StatusRequestEntityTooLarge, body, nil); err != nil {
		return fail("payload_limit", err)
	}
	return ok("payload_limit")
}

func rateLimitGateway(ctx context.Context, c *http.Client, base string) result {
	// flood /auth/healthz; expect at least one 429
	got429 := false
	for i := 0; i < 400; i++ {
		code, _, err := doRequest(ctx, c, base+"/auth/healthz", http.MethodGet, nil, nil)
		if err != nil {
			return fail("rate_limit", err)
		}
		if code == http.StatusTooManyRequests {
			got429 = true
			break
		}
	}
	if !got429 {
		return fail("rate_limit", errors.New("no 429 observed"))
	}
	return ok("rate_limit")
}

func authRegisterLoginLogout(ctx context.Context, c *http.Client, base string) result {
	email := randomEmail()
	password := "StrongP@ss1"

	// register
	reg := fmt.Sprintf(`{"email":"%s","username":"%s","password":"%s"}`, email, strings.Split(email, "@")[0], password)
	if err := expectStatus(ctx, c, base+"/auth/register", http.StatusCreated, []byte(reg), jsonHeader()); err != nil {
		return fail("auth_register", err)
	}
	if err := verifyEmail(ctx, c, email); err != nil {
		return fail("auth_verify", err)
	}

	// login
	login := fmt.Sprintf(`{"identifier":"%s","password":"%s"}`, email, password)
	if err := expectStatus(ctx, c, base+"/auth/login", http.StatusOK, []byte(login), jsonHeader()); err != nil {
		return fail("auth_login", err)
	}
	csrf := findCookie(c, base+"/auth", "csrf_token")
	if csrf == "" {
		return fail("auth_login", errors.New("csrf cookie missing"))
	}

	// me
	if err := expectStatus(ctx, c, base+"/auth/me", http.StatusOK, nil, nil); err != nil {
		return fail("auth_me", err)
	}

	// logout without csrf -> 403
	code, _, err := doRequest(ctx, c, base+"/auth/logout", http.MethodPost, nil, nil)
	if err != nil || code != http.StatusForbidden {
		return fail("auth_logout_csrf_missing", fmt.Errorf("got %d err=%v", code, err))
	}

	// logout with csrf -> 204
	hdr := jsonHeader()
	hdr["X-CSRF-Token"] = csrf
	if err := expectStatusWithMethod(ctx, c, base+"/auth/logout", http.MethodPost, http.StatusNoContent, nil, hdr); err != nil {
		return fail("auth_logout", err)
	}

	// session should be gone
	codeAfter, _, err := doRequest(ctx, c, base+"/auth/me", http.MethodGet, nil, nil)
	if err != nil {
		return fail("auth_logout_me", err)
	}
	if codeAfter != http.StatusUnauthorized {
		return fail("auth_logout_me", fmt.Errorf("expected 401 after logout, got %d", codeAfter))
	}

	return ok("auth_basic_flow")
}

func authMFAFlow(ctx context.Context, c *http.Client, base string) result {
	email := randomEmail()
	password := "AnotherP@ss1"

	reg := fmt.Sprintf(`{"email":"%s","username":"%s","password":"%s"}`, email, strings.Split(email, "@")[0], password)
	if err := expectStatus(ctx, c, base+"/auth/register", http.StatusCreated, []byte(reg), jsonHeader()); err != nil {
		return fail("mfa_register", err)
	}
	if err := verifyEmail(ctx, c, email); err != nil {
		return fail("mfa_verify_email", err)
	}

	login := fmt.Sprintf(`{"identifier":"%s","password":"%s"}`, email, password)
	if err := expectStatus(ctx, c, base+"/auth/login", http.StatusOK, []byte(login), jsonHeader()); err != nil {
		return fail("mfa_login", err)
	}
	csrf := findCookie(c, base+"/auth", "csrf_token")
	if csrf == "" {
		return fail("mfa_login", errors.New("csrf missing"))
	}

	// setup TOTP
	hdr := jsonHeader()
	hdr["X-CSRF-Token"] = csrf
	status, body, err := doJSON(ctx, c, base+"/auth/mfa/totp/setup", http.MethodPost, nil, hdr)
	if err != nil || status != http.StatusOK {
		return fail("mfa_setup", fmt.Errorf("status %d err %v", status, err))
	}
	var setup struct {
		Secret string `json:"secret"`
	}
	if err := json.Unmarshal(body, &setup); err != nil || setup.Secret == "" {
		return fail("mfa_setup", errors.New("missing secret"))
	}

	code, err := totpCode(setup.Secret, time.Now(), 30)
	if err != nil {
		return fail("mfa_code", err)
	}
	verify := fmt.Sprintf(`{"code":"%s"}`, code)
	status, body, err = doJSON(ctx, c, base+"/auth/mfa/totp/verify", http.MethodPost, []byte(verify), hdr)
	if err != nil || status != http.StatusOK {
		return fail("mfa_verify", fmt.Errorf("status %d err %v", status, err))
	}
	var vr struct {
		BackupCodes []string `json:"backup_codes"`
	}
	_ = json.Unmarshal(body, &vr)
	if len(vr.BackupCodes) == 0 {
		return fail("mfa_verify", errors.New("no backup codes"))
	}

	// logout
	if err := expectStatusWithMethod(ctx, c, base+"/auth/logout", http.MethodPost, http.StatusNoContent, nil, hdr); err != nil {
		return fail("mfa_logout", err)
	}

	// login without totp -> expect 401
	codeStatus, _, err := doJSON(ctx, c, base+"/auth/login", http.MethodPost, []byte(login), jsonHeader())
	if err != nil || codeStatus != http.StatusUnauthorized {
		return fail("mfa_login_requires_totp", fmt.Errorf("status %d err %v", codeStatus, err))
	}

	// login with totp
	totpLogin := fmt.Sprintf(`{"identifier":"%s","password":"%s","totp_code":"%s"}`, email, password, code)
	if err := expectStatus(ctx, c, base+"/auth/login", http.StatusOK, []byte(totpLogin), jsonHeader()); err != nil {
		return fail("mfa_login_totp", err)
	}
	csrf = findCookie(c, base+"/auth", "csrf_token")
	hdr["X-CSRF-Token"] = csrf

	// logout
	if err := expectStatusWithMethod(ctx, c, base+"/auth/logout", http.MethodPost, http.StatusNoContent, nil, hdr); err != nil {
		return fail("mfa_logout2", err)
	}

	// login with backup code (single use)
	backup := vr.BackupCodes[0]
	backupLogin := fmt.Sprintf(`{"identifier":"%s","password":"%s","backup_code":"%s"}`, email, password, backup)
	if err := expectStatus(ctx, c, base+"/auth/login", http.StatusOK, []byte(backupLogin), jsonHeader()); err != nil {
		return fail("mfa_backup_login", err)
	}
	csrf = findCookie(c, base+"/auth", "csrf_token")
	hdr["X-CSRF-Token"] = csrf
	_ = expectStatus(ctx, c, base+"/auth/logout", http.StatusNoContent, nil, hdr)

	// backup code should now be spent
	codeStatus, _, err = doJSON(ctx, c, base+"/auth/login", http.MethodPost, []byte(backupLogin), jsonHeader())
	if err == nil && codeStatus == http.StatusOK {
		return fail("mfa_backup_single_use", errors.New("backup code reused successfully"))
	}

	return ok("mfa_flow")
}

// Extra CSRF checks on other mutating endpoints
func authCSRFExtra(ctx context.Context, c *http.Client, base string) result {
	email := randomEmail()
	password := "CsrfExtra1!"

	reg := fmt.Sprintf(`{"email":"%s","username":"%s","password":"%s"}`, email, strings.Split(email, "@")[0], password)
	if err := expectStatus(ctx, c, base+"/auth/register", http.StatusCreated, []byte(reg), jsonHeader()); err != nil {
		return fail("csrf_extra_register", err)
	}
	if err := verifyEmail(ctx, c, email); err != nil {
		return fail("csrf_extra_verify_email", err)
	}

	login := fmt.Sprintf(`{"identifier":"%s","password":"%s"}`, email, password)
	if err := expectStatus(ctx, c, base+"/auth/login", http.StatusOK, []byte(login), jsonHeader()); err != nil {
		return fail("csrf_extra_login", err)
	}
	csrf := findCookie(c, base+"/auth", "csrf_token")
	if csrf == "" {
		return fail("csrf_extra_login", errors.New("csrf missing"))
	}

	// regenerate backup without csrf -> expect 403
	code, _, err := doRequest(ctx, c, base+"/auth/mfa/backup/regenerate", http.MethodPost, nil, nil)
	if err != nil {
		return fail("csrf_extra_regen_no_csrf", err)
	}
	if code != http.StatusForbidden {
		return fail("csrf_extra_regen_no_csrf", fmt.Errorf("expected 403, got %d", code))
	}

	// regenerate backup with csrf -> expect 200 (even if no MFA configured yet, auth should answer deterministically)
	hdr := jsonHeader()
	hdr["X-CSRF-Token"] = csrf
	code, _, err = doJSON(ctx, c, base+"/auth/mfa/backup/regenerate", http.MethodPost, nil, hdr)
	if err != nil || (code != http.StatusOK && code != http.StatusBadRequest && code != http.StatusUnauthorized) {
		return fail("csrf_extra_regen_csrf", fmt.Errorf("unexpected status %d err %v", code, err))
	}

	// logout with csrf
	if err := expectStatusWithMethod(ctx, c, base+"/auth/logout", http.MethodPost, http.StatusNoContent, nil, hdr); err != nil {
		return fail("csrf_extra_logout", err)
	}

	return ok("csrf_extra")
}

// ---- Helpers ----

func expectStatus(ctx context.Context, c *http.Client, url string, want int, body []byte, headers map[string]string) error {
	return expectStatusWithMethod(ctx, c, url, methodForBody(body), want, body, headers)
}

func expectStatusWithMethod(ctx context.Context, c *http.Client, url, method string, want int, body []byte, headers map[string]string) error {
	code, _, err := doJSON(ctx, c, url, method, body, headers)
	if err != nil {
		return err
	}
	if code != want {
		return fmt.Errorf("unexpected status %d want %d", code, want)
	}
	return nil
}

func doJSON(ctx context.Context, c *http.Client, url, method string, body []byte, headers map[string]string) (int, []byte, error) {
	if headers == nil {
		headers = make(map[string]string)
	}
	if body != nil && headers["Content-Type"] == "" {
		headers["Content-Type"] = "application/json"
	}
	code, data, err := doRequest(ctx, c, url, method, body, headers)
	return code, data, err
}

func doRequest(ctx context.Context, c *http.Client, url, method string, body []byte, headers map[string]string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return resp.StatusCode, data, nil
}

func methodForBody(body []byte) string {
	if body == nil {
		return http.MethodGet
	}
	return http.MethodPost
}

func ok(name string) result { return result{Name: name, Status: "ok"} }
func fail(name string, err error) result {
	return result{Name: name, Status: "fail", Detail: err.Error()}
}

func randomEmail() string {
	return fmt.Sprintf("user%d@example.com", time.Now().UnixNano()+int64(rand.Intn(1000)))
}

func findCookie(c *http.Client, base, name string) string {
	u := base
	if !strings.HasPrefix(u, "http") {
		u = "http://" + strings.TrimLeft(u, "/")
	}
	parsed, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return ""
	}
	cookies := c.Jar.Cookies(parsed.URL)
	for _, ck := range cookies {
		if ck.Name == name {
			return ck.Value
		}
	}
	return ""
}

func jsonHeader() map[string]string {
	return map[string]string{
		"Content-Type":    "application/json",
		"X-Client-Family": "cli",
	}
}

func verifyEmail(ctx context.Context, c *http.Client, email string) error {
	authBase := strings.TrimRight(getenv("AUTH_BASE_URL", "http://auth:8080"), "/")
	issueToken := strings.TrimSpace(os.Getenv("AUTH_EMAIL_VERIFICATION_INTERNAL_TOKEN"))
	outboxDir := strings.TrimSpace(getenv("AUTH_EMAIL_OUTBOX_DIR", getenv("EMAIL_OUTBOX_DIR", "")))
	if outboxDir != "" {
		token, err := waitForOutboxToken(outboxDir, "verify", email)
		if err != nil {
			return err
		}
		confirmBody := []byte(fmt.Sprintf(`{"token":"%s"}`, token))
		code, _, err := doJSON(ctx, c, authBase+"/email/verify/confirm", http.MethodPost, confirmBody, jsonHeader())
		if err != nil {
			return err
		}
		if code != http.StatusOK {
			return fmt.Errorf("confirm verification returned %d", code)
		}
		return nil
	}
	if issueToken == "" {
		return errors.New("AUTH_EMAIL_OUTBOX_DIR or AUTH_EMAIL_VERIFICATION_INTERNAL_TOKEN required")
	}

	body := []byte(fmt.Sprintf(`{"email":"%s"}`, email))
	headers := jsonHeader()
	headers["X-Internal-Token"] = issueToken
	code, payload, err := doJSON(ctx, c, authBase+"/internal/email-verifications/issue", http.MethodPost, body, headers)
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("issue verification token returned %d", code)
	}
	var issued struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(payload, &issued); err != nil {
		return err
	}
	if strings.TrimSpace(issued.Token) == "" {
		return errors.New("verification token missing")
	}

	confirmBody := []byte(fmt.Sprintf(`{"token":"%s"}`, issued.Token))
	code, _, err = doJSON(ctx, c, authBase+"/email/verify/confirm", http.MethodPost, confirmBody, jsonHeader())
	if err != nil {
		return err
	}
	if code != http.StatusOK {
		return fmt.Errorf("confirm verification returned %d", code)
	}
	return nil
}

func waitForOutboxToken(dir, prefix, email string) (string, error) {
	want := sanitizeOutboxRecipient(email)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		entries, err := os.ReadDir(dir)
		if err == nil {
			for i := len(entries) - 1; i >= 0; i-- {
				name := entries[i].Name()
				if entries[i].IsDir() || !strings.HasPrefix(name, prefix) || !strings.Contains(name, want) {
					continue
				}
				payload, err := os.ReadFile(filepath.Join(dir, name))
				if err != nil {
					continue
				}
				token := extractLastTokenLine(string(payload))
				if token != "" {
					return token, nil
				}
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return "", fmt.Errorf("token with prefix %s for %s not found in outbox %s", prefix, email, dir)
}

func sanitizeOutboxRecipient(email string) string {
	email = strings.TrimSpace(strings.ToLower(email))
	replacer := strings.NewReplacer("@", "_at_", "/", "_", "\\", "_", ":", "_", " ", "_")
	return replacer.Replace(email)
}

func extractLastTokenLine(payload string) string {
	lines := strings.Split(payload, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "submit it to post ") {
			continue
		}
		if strings.HasPrefix(strings.ToLower(line), "use this ") {
			continue
		}
		return line
	}
	return ""
}

// totpCode implements RFC6238 (SHA1, 6 digits, period provided)
func totpCode(secret string, t time.Time, periodSeconds int) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", err
	}
	counter := uint64(t.Unix() / int64(periodSeconds))
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)
	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0F
	codeInt := (uint32(sum[offset])&0x7F)<<24 | (uint32(sum[offset+1])&0xFF)<<16 | (uint32(sum[offset+2])&0xFF)<<8 | (uint32(sum[offset+3]) & 0xFF)
	codeInt = codeInt % 1_000_000
	return fmt.Sprintf("%06d", codeInt), nil
}
