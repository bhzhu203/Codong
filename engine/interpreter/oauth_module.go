package interpreter

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/codong-lang/codong/stdlib/codongerror"
)

// OAuthModuleObject is the singleton `oauth` module.
type OAuthModuleObject struct {
	providers map[string]*oauthProvider
	jwtConfig *jwtConfig
	roles     map[string][]string // RBAC roles → permissions
	mu        sync.RWMutex
}

type oauthProvider struct {
	Name         string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       []string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	TenantID     string // Microsoft
}

type jwtConfig struct {
	Secret            string
	Algorithm         string
	ExpiresIn         time.Duration
	RefreshExpiresIn  time.Duration
	IncludeJTI        bool
}

func (o *OAuthModuleObject) Type() string    { return "module" }
func (o *OAuthModuleObject) Inspect() string { return "<module:oauth>" }

var oauthModuleSingleton = &OAuthModuleObject{
	providers: make(map[string]*oauthProvider),
	roles:     make(map[string][]string),
}

func oauthError(code, message, fix string) Object {
	return &ErrorObject{
		Error:     codongerror.New(code, message, codongerror.WithFix(fix)),
		IsRuntime: true,
	}
}

// Known provider configurations
var knownProviders = map[string]struct {
	AuthURL     string
	TokenURL    string
	UserInfoURL string
}{
	"github": {
		AuthURL:     "https://github.com/login/oauth/authorize",
		TokenURL:    "https://github.com/login/oauth/access_token",
		UserInfoURL: "https://api.github.com/user",
	},
	"google": {
		AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:    "https://oauth2.googleapis.com/token",
		UserInfoURL: "https://openidconnect.googleapis.com/v1/userinfo",
	},
	"microsoft": {
		AuthURL:     "https://login.microsoftonline.com/{tenant}/oauth2/v2.0/authorize",
		TokenURL:    "https://login.microsoftonline.com/{tenant}/oauth2/v2.0/token",
		UserInfoURL: "https://graph.microsoft.com/v1.0/me",
	},
}

// evalOAuthModuleMethod dispatches oauth.xxx() calls.
func (interp *Interpreter) evalOAuthModuleMethod(method string) Object {
	return &BuiltinFunction{
		Name: "oauth." + method,
		Fn: func(i *Interpreter, args ...Object) Object {
			switch method {
			case "provider":
				return i.oauthProvider(args)
			case "configure_jwt":
				return i.oauthConfigureJWT(args)
			case "authorization_url":
				return i.oauthAuthorizationURL(args)
			case "exchange_code":
				return i.oauthExchangeCode(args)
			case "get_profile":
				return i.oauthGetProfile(args)
			case "sign_jwt":
				return i.oauthSignJWT(args)
			case "sign_refresh_token":
				return i.oauthSignRefreshToken(args)
			case "verify_jwt":
				return i.oauthVerifyJWT(args)
			case "verify_refresh_token":
				return i.oauthVerifyJWT(args)
			case "decode_jwt":
				return i.oauthDecodeJWT(args)
			case "revoke_jwt":
				return i.oauthRevokeJWT(args)
			case "is_revoked":
				return i.oauthIsRevoked(args)
			case "generate_state":
				return i.oauthGenerateState()
			case "generate_pkce":
				return i.oauthGeneratePKCE()
			case "hash_token":
				return i.oauthHashToken(args)
			case "define_roles":
				return i.oauthDefineRoles(args)
			case "has_permission":
				return i.oauthHasPermission(args)
			case "check_permission":
				return i.oauthHasPermission(args)
			default:
				return oauthError(codongerror.E14007_PROVIDER_ERROR,
					fmt.Sprintf("unknown oauth method: %s", method), "")
			}
		},
	}
}

func (i *Interpreter) oauthProvider(args []Object) Object {
	if len(args) < 2 {
		return oauthError(codongerror.E14007_PROVIDER_ERROR,
			"oauth.provider requires (name, config)", "")
	}

	name := args[0].Inspect()
	config, ok := args[1].(*MapObject)
	if !ok {
		return oauthError(codongerror.E14007_PROVIDER_ERROR, "config must be a map", "")
	}

	p := &oauthProvider{Name: name}

	if v, ok := config.Entries["client_id"]; ok {
		p.ClientID = v.Inspect()
	}
	if v, ok := config.Entries["client_secret"]; ok {
		p.ClientSecret = v.Inspect()
	}
	if v, ok := config.Entries["redirect_uri"]; ok {
		p.RedirectURI = v.Inspect()
	}
	if v, ok := config.Entries["tenant_id"]; ok {
		p.TenantID = v.Inspect()
	}
	if v, ok := config.Entries["scopes"]; ok {
		if list, ok := v.(*ListObject); ok {
			for _, s := range list.Elements {
				p.Scopes = append(p.Scopes, s.Inspect())
			}
		}
	}

	// Set URLs from known providers
	if known, ok := knownProviders[name]; ok {
		p.AuthURL = known.AuthURL
		p.TokenURL = known.TokenURL
		p.UserInfoURL = known.UserInfoURL

		// Microsoft tenant substitution
		if name == "microsoft" && p.TenantID != "" {
			p.AuthURL = strings.ReplaceAll(p.AuthURL, "{tenant}", p.TenantID)
			p.TokenURL = strings.ReplaceAll(p.TokenURL, "{tenant}", p.TenantID)
		} else if name == "microsoft" {
			p.AuthURL = strings.ReplaceAll(p.AuthURL, "{tenant}", "common")
			p.TokenURL = strings.ReplaceAll(p.TokenURL, "{tenant}", "common")
		}
	}

	// Allow overriding URLs
	if v, ok := config.Entries["auth_url"]; ok {
		p.AuthURL = v.Inspect()
	}
	if v, ok := config.Entries["token_url"]; ok {
		p.TokenURL = v.Inspect()
	}
	if v, ok := config.Entries["userinfo_url"]; ok {
		p.UserInfoURL = v.Inspect()
	}

	oauthModuleSingleton.mu.Lock()
	oauthModuleSingleton.providers[name] = p
	oauthModuleSingleton.mu.Unlock()

	return NULL_OBJ
}

func (i *Interpreter) oauthConfigureJWT(args []Object) Object {
	if len(args) < 1 {
		return oauthError(codongerror.E14003_INVALID_TOKEN, "configure_jwt requires config", "")
	}
	config, ok := args[0].(*MapObject)
	if !ok {
		return oauthError(codongerror.E14003_INVALID_TOKEN, "config must be a map", "")
	}

	jc := &jwtConfig{
		Algorithm:        "HS256",
		ExpiresIn:        24 * time.Hour,
		RefreshExpiresIn: 30 * 24 * time.Hour,
	}

	if v, ok := config.Entries["secret"]; ok {
		jc.Secret = v.Inspect()
	}
	if v, ok := config.Entries["algorithm"]; ok {
		jc.Algorithm = v.Inspect()
	}
	if v, ok := config.Entries["expires_in"]; ok {
		if d, err := time.ParseDuration(v.Inspect()); err == nil {
			jc.ExpiresIn = d
		}
	}
	if v, ok := config.Entries["refresh_expires_in"]; ok {
		if d, err := time.ParseDuration(v.Inspect()); err == nil {
			jc.RefreshExpiresIn = d
		}
	}
	if v, ok := config.Entries["include_jti"]; ok {
		if b, ok := v.(*BoolObject); ok {
			jc.IncludeJTI = b.Value
		}
	}

	oauthModuleSingleton.jwtConfig = jc
	return NULL_OBJ
}

func (i *Interpreter) oauthAuthorizationURL(args []Object) Object {
	if len(args) < 1 {
		return oauthError(codongerror.E14007_PROVIDER_ERROR,
			"authorization_url requires a provider name", "")
	}

	providerName := args[0].Inspect()
	oauthModuleSingleton.mu.RLock()
	p, ok := oauthModuleSingleton.providers[providerName]
	oauthModuleSingleton.mu.RUnlock()
	if !ok {
		return oauthError(codongerror.E14007_PROVIDER_ERROR,
			fmt.Sprintf("provider '%s' not configured", providerName),
			"call oauth.provider() first")
	}

	// Build authorization URL
	u, _ := url.Parse(p.AuthURL)
	q := u.Query()
	q.Set("client_id", p.ClientID)
	q.Set("redirect_uri", p.RedirectURI)
	q.Set("response_type", "code")
	if len(p.Scopes) > 0 {
		q.Set("scope", strings.Join(p.Scopes, " "))
	}

	// Parse named args for state and PKCE
	for _, a := range args[1:] {
		if m, ok := a.(*MapObject); ok {
			if v, ok := m.Entries["state"]; ok {
				q.Set("state", v.Inspect())
			}
			if v, ok := m.Entries["code_challenge"]; ok {
				q.Set("code_challenge", v.Inspect())
			}
			if v, ok := m.Entries["code_challenge_method"]; ok {
				q.Set("code_challenge_method", v.Inspect())
			}
		}
	}

	u.RawQuery = q.Encode()
	return &StringObject{Value: u.String()}
}

func (i *Interpreter) oauthExchangeCode(args []Object) Object {
	if len(args) < 2 {
		return oauthError(codongerror.E14002_CODE_EXCHANGE_FAILED,
			"exchange_code requires (provider, code)", "")
	}

	providerName := args[0].Inspect()
	code := args[1].Inspect()

	oauthModuleSingleton.mu.RLock()
	p, ok := oauthModuleSingleton.providers[providerName]
	oauthModuleSingleton.mu.RUnlock()
	if !ok {
		return oauthError(codongerror.E14007_PROVIDER_ERROR,
			fmt.Sprintf("provider '%s' not configured", providerName), "")
	}

	// Build token exchange request
	params := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {p.RedirectURI},
		"client_id":     {p.ClientID},
		"client_secret": {p.ClientSecret},
	}

	// Check for code_verifier in named args (PKCE)
	for _, a := range args[2:] {
		if m, ok := a.(*MapObject); ok {
			if v, ok := m.Entries["code_verifier"]; ok {
				params.Set("code_verifier", v.Inspect())
			}
		}
	}

	// Make HTTP request
	req, _ := http.NewRequest("POST", p.TokenURL, strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return oauthError(codongerror.E14002_CODE_EXCHANGE_FAILED,
			fmt.Sprintf("token exchange failed: %s", err.Error()), "")
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var tokenResp map[string]interface{}
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		// Try URL-encoded format (GitHub)
		vals, err2 := url.ParseQuery(string(body))
		if err2 != nil {
			return oauthError(codongerror.E14002_CODE_EXCHANGE_FAILED,
				"failed to parse token response", "")
		}
		tokenResp = make(map[string]interface{})
		for k, v := range vals {
			if len(v) > 0 {
				tokenResp[k] = v[0]
			}
		}
	}

	if errMsg, ok := tokenResp["error"]; ok {
		return oauthError(codongerror.E14002_CODE_EXCHANGE_FAILED,
			fmt.Sprintf("provider error: %v", errMsg), "code may be expired")
	}

	result := &MapObject{Entries: map[string]Object{}, Order: []string{}}
	for k, v := range tokenResp {
		result.Entries[k] = &StringObject{Value: fmt.Sprintf("%v", v)}
		result.Order = append(result.Order, k)
	}

	return result
}

func (i *Interpreter) oauthGetProfile(args []Object) Object {
	if len(args) < 2 {
		return oauthError(codongerror.E14008_PROFILE_FETCH_FAILED,
			"get_profile requires (provider, access_token)", "")
	}

	providerName := args[0].Inspect()
	accessToken := args[1].Inspect()

	oauthModuleSingleton.mu.RLock()
	p, ok := oauthModuleSingleton.providers[providerName]
	oauthModuleSingleton.mu.RUnlock()
	if !ok {
		return oauthError(codongerror.E14007_PROVIDER_ERROR,
			fmt.Sprintf("provider '%s' not configured", providerName), "")
	}

	req, _ := http.NewRequest("GET", p.UserInfoURL, nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	// GitHub needs User-Agent
	if providerName == "github" {
		req.Header.Set("User-Agent", "Codong-OAuth")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return oauthError(codongerror.E14008_PROFILE_FETCH_FAILED,
			fmt.Sprintf("profile fetch failed: %s", err.Error()), "")
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var profile map[string]interface{}
	if err := json.Unmarshal(body, &profile); err != nil {
		return oauthError(codongerror.E14008_PROFILE_FETCH_FAILED,
			"failed to parse profile response", "")
	}

	// Normalize to standard format
	result := &MapObject{Entries: map[string]Object{}, Order: []string{}}
	result.Entries["provider"] = &StringObject{Value: providerName}
	result.Order = append(result.Order, "provider")

	switch providerName {
	case "github":
		setProfileField(result, "id", profile, "id")
		setProfileField(result, "email", profile, "email")
		setProfileField(result, "name", profile, "name")
		setProfileField(result, "avatar", profile, "avatar_url")
		setProfileField(result, "login", profile, "login")
	case "google":
		setProfileField(result, "id", profile, "sub")
		setProfileField(result, "email", profile, "email")
		setProfileField(result, "name", profile, "name")
		setProfileField(result, "given_name", profile, "given_name")
		setProfileField(result, "family_name", profile, "family_name")
		setProfileField(result, "avatar", profile, "picture")
		setProfileField(result, "locale", profile, "locale")
	case "microsoft":
		setProfileField(result, "id", profile, "id")
		setProfileField(result, "email", profile, "mail")
		setProfileField(result, "name", profile, "displayName")
		setProfileField(result, "given_name", profile, "givenName")
		setProfileField(result, "family_name", profile, "surname")
	default:
		// Generic: pass through all fields
		for k, v := range profile {
			result.Entries[k] = &StringObject{Value: fmt.Sprintf("%v", v)}
			result.Order = append(result.Order, k)
		}
	}

	// Store raw response
	rawMap := &MapObject{Entries: map[string]Object{}, Order: []string{}}
	for k, v := range profile {
		rawMap.Entries[k] = &StringObject{Value: fmt.Sprintf("%v", v)}
		rawMap.Order = append(rawMap.Order, k)
	}
	result.Entries["raw"] = rawMap
	result.Order = append(result.Order, "raw")

	return result
}

func setProfileField(result *MapObject, key string, profile map[string]interface{}, sourceKey string) {
	if v, ok := profile[sourceKey]; ok && v != nil {
		result.Entries[key] = &StringObject{Value: fmt.Sprintf("%v", v)}
	} else {
		result.Entries[key] = NULL_OBJ
	}
	result.Order = append(result.Order, key)
}

// JWT implementation (HS256)

func (i *Interpreter) oauthSignJWT(args []Object) Object {
	jc := oauthModuleSingleton.jwtConfig
	if jc == nil {
		jc = &jwtConfig{Secret: "codong-default-secret", ExpiresIn: 24 * time.Hour}
	}

	if len(args) < 1 {
		return oauthError(codongerror.E14003_INVALID_TOKEN, "sign_jwt requires claims", "")
	}

	claims, ok := args[0].(*MapObject)
	if !ok {
		return oauthError(codongerror.E14003_INVALID_TOKEN, "claims must be a map", "")
	}

	// Build JWT payload
	payload := make(map[string]interface{})
	for _, k := range claims.Order {
		v := claims.Entries[k]
		switch obj := v.(type) {
		case *StringObject:
			payload[k] = obj.Value
		case *NumberObject:
			payload[k] = obj.Value
		case *BoolObject:
			payload[k] = obj.Value
		case *ListObject:
			arr := make([]interface{}, len(obj.Elements))
			for j, el := range obj.Elements {
				arr[j] = el.Inspect()
			}
			payload[k] = arr
		default:
			payload[k] = v.Inspect()
		}
	}

	now := time.Now()
	expiresIn := jc.ExpiresIn

	// Check for overriding expires_in in named args
	for _, a := range args[1:] {
		if m, ok := a.(*MapObject); ok {
			if v, ok := m.Entries["expires_in"]; ok {
				if d, err := time.ParseDuration(v.Inspect()); err == nil {
					expiresIn = d
				}
			}
		}
	}

	payload["iat"] = now.Unix()
	payload["exp"] = now.Add(expiresIn).Unix()

	if jc.IncludeJTI {
		b := make([]byte, 16)
		rand.Read(b)
		payload["jti"] = hex.EncodeToString(b)
	}

	token := signHS256(payload, jc.Secret)
	return &StringObject{Value: token}
}

func (i *Interpreter) oauthSignRefreshToken(args []Object) Object {
	jc := oauthModuleSingleton.jwtConfig
	if jc == nil {
		jc = &jwtConfig{Secret: "codong-default-secret", RefreshExpiresIn: 30 * 24 * time.Hour}
	}

	if len(args) < 1 {
		return oauthError(codongerror.E14003_INVALID_TOKEN, "sign_refresh_token requires claims", "")
	}

	claims, ok := args[0].(*MapObject)
	if !ok {
		return oauthError(codongerror.E14003_INVALID_TOKEN, "claims must be a map", "")
	}

	payload := make(map[string]interface{})
	for _, k := range claims.Order {
		payload[k] = claims.Entries[k].Inspect()
	}

	now := time.Now()
	payload["iat"] = now.Unix()
	payload["exp"] = now.Add(jc.RefreshExpiresIn).Unix()
	payload["type"] = "refresh"

	token := signHS256(payload, jc.Secret)
	return &StringObject{Value: token}
}

func (i *Interpreter) oauthVerifyJWT(args []Object) Object {
	jc := oauthModuleSingleton.jwtConfig
	if jc == nil {
		jc = &jwtConfig{Secret: "codong-default-secret"}
	}

	if len(args) < 1 {
		return NULL_OBJ
	}

	token := args[0].Inspect()
	claims, err := verifyHS256(token, jc.Secret)
	if err != nil {
		return NULL_OBJ
	}

	// Check expiration
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return NULL_OBJ
		}
	}

	result := &MapObject{Entries: map[string]Object{}, Order: []string{}}
	for k, v := range claims {
		switch val := v.(type) {
		case float64:
			result.Entries[k] = &NumberObject{Value: val}
		case string:
			result.Entries[k] = &StringObject{Value: val}
		case bool:
			result.Entries[k] = nativeBoolToObject(val)
		case []interface{}:
			elements := make([]Object, len(val))
			for j, el := range val {
				elements[j] = &StringObject{Value: fmt.Sprintf("%v", el)}
			}
			result.Entries[k] = &ListObject{Elements: elements}
		default:
			result.Entries[k] = &StringObject{Value: fmt.Sprintf("%v", v)}
		}
		result.Order = append(result.Order, k)
	}
	return result
}

func (i *Interpreter) oauthDecodeJWT(args []Object) Object {
	if len(args) < 1 {
		return NULL_OBJ
	}
	token := args[0].Inspect()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return NULL_OBJ
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return NULL_OBJ
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return NULL_OBJ
	}

	result := &MapObject{Entries: map[string]Object{}, Order: []string{}}
	for k, v := range claims {
		result.Entries[k] = &StringObject{Value: fmt.Sprintf("%v", v)}
		result.Order = append(result.Order, k)
	}
	return result
}

// LRU cache for revoked tokens
var (
	revokedTokens     = make(map[string]int64) // jti → expiry timestamp
	revokedTokensMu   sync.RWMutex
	revokedMaxEntries = 10000
)

func (i *Interpreter) oauthRevokeJWT(args []Object) Object {
	if len(args) < 1 {
		return NULL_OBJ
	}

	token := args[0].Inspect()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return NULL_OBJ
	}

	payloadBytes, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var claims map[string]interface{}
	json.Unmarshal(payloadBytes, &claims)

	jti, _ := claims["jti"].(string)
	if jti == "" {
		// Use token hash as key
		h := sha256.Sum256([]byte(token))
		jti = hex.EncodeToString(h[:])
	}

	exp := int64(0)
	if e, ok := claims["exp"].(float64); ok {
		exp = int64(e)
	}

	revokedTokensMu.Lock()
	revokedTokens[jti] = exp

	// Evict expired entries if over limit
	if len(revokedTokens) > revokedMaxEntries {
		now := time.Now().Unix()
		for k, v := range revokedTokens {
			if v > 0 && v < now {
				delete(revokedTokens, k)
			}
		}
	}
	revokedTokensMu.Unlock()

	return NULL_OBJ
}

func (i *Interpreter) oauthIsRevoked(args []Object) Object {
	if len(args) < 1 {
		return FALSE_OBJ
	}

	token := args[0].Inspect()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return FALSE_OBJ
	}

	payloadBytes, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var claims map[string]interface{}
	json.Unmarshal(payloadBytes, &claims)

	jti, _ := claims["jti"].(string)
	if jti == "" {
		h := sha256.Sum256([]byte(token))
		jti = hex.EncodeToString(h[:])
	}

	revokedTokensMu.RLock()
	_, revoked := revokedTokens[jti]
	revokedTokensMu.RUnlock()

	return nativeBoolToObject(revoked)
}

func (i *Interpreter) oauthGenerateState() Object {
	b := make([]byte, 32)
	rand.Read(b)
	return &StringObject{Value: hex.EncodeToString(b)}
}

func (i *Interpreter) oauthGeneratePKCE() Object {
	verifierBytes := make([]byte, 32)
	rand.Read(verifierBytes)
	verifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	h := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(h[:])

	return &MapObject{
		Entries: map[string]Object{
			"code_verifier":  &StringObject{Value: verifier},
			"code_challenge": &StringObject{Value: challenge},
			"method":         &StringObject{Value: "S256"},
		},
		Order: []string{"code_verifier", "code_challenge", "method"},
	}
}

func (i *Interpreter) oauthHashToken(args []Object) Object {
	if len(args) < 1 {
		return &StringObject{Value: ""}
	}
	h := sha256.Sum256([]byte(args[0].Inspect()))
	return &StringObject{Value: hex.EncodeToString(h[:])}
}

func (i *Interpreter) oauthDefineRoles(args []Object) Object {
	if len(args) < 1 {
		return NULL_OBJ
	}
	roles, ok := args[0].(*MapObject)
	if !ok {
		return NULL_OBJ
	}

	oauthModuleSingleton.mu.Lock()
	for _, k := range roles.Order {
		v := roles.Entries[k]
		if list, ok := v.(*ListObject); ok {
			perms := make([]string, len(list.Elements))
			for j, el := range list.Elements {
				perms[j] = el.Inspect()
			}
			oauthModuleSingleton.roles[k] = perms
		}
	}
	oauthModuleSingleton.mu.Unlock()
	return NULL_OBJ
}

func (i *Interpreter) oauthHasPermission(args []Object) Object {
	if len(args) < 2 {
		return FALSE_OBJ
	}

	// First arg: roles list or user object
	var userRoles []string
	if list, ok := args[0].(*ListObject); ok {
		for _, el := range list.Elements {
			userRoles = append(userRoles, el.Inspect())
		}
	}

	permission := args[1].Inspect()

	oauthModuleSingleton.mu.RLock()
	defer oauthModuleSingleton.mu.RUnlock()

	for _, role := range userRoles {
		if perms, ok := oauthModuleSingleton.roles[role]; ok {
			for _, p := range perms {
				if p == permission {
					return TRUE_OBJ
				}
				// Wildcard: "users:*" matches "users:read"
				if strings.HasSuffix(p, ":*") {
					prefix := strings.TrimSuffix(p, "*")
					if strings.HasPrefix(permission, prefix) {
						return TRUE_OBJ
					}
				}
			}
		}
	}
	return FALSE_OBJ
}

// HS256 JWT helper functions

func signHS256(payload map[string]interface{}, secret string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	payloadBytes, _ := json.Marshal(payload)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadBytes)

	signingInput := header + "." + payloadB64

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signingInput + "." + signature
}

func verifyHS256(token, secret string) (map[string]interface{}, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	signingInput := parts[0] + "." + parts[1]

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, fmt.Errorf("invalid signature")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, err
	}

	return claims, nil
}
