package sepay

import "fmt"

const (
	// EnvironmentProduction selects the live SePay User API.
	EnvironmentProduction = "production"
	// EnvironmentSandbox selects the isolated SePay Test Mode API.
	EnvironmentSandbox = "sandbox"

	AuthMethodHMAC   = "hmac"
	AuthMethodAPIKey = "apikey"
	AuthMethodNone   = "none"
)

type Credentials struct {
	Environment       string `json:"environment"`
	BankShortName     string `json:"bank_short_name"`
	AccountNumber     string `json:"account_number"`
	AccountName       string `json:"account_name"`
	CodePrefix        string `json:"code_prefix"`
	WebhookAuthMethod string `json:"webhook_auth_method"` // "apikey" | "hmac" | "none"
	WebhookAPIKey     string `json:"webhook_api_key"`
	WebhookSecret     string `json:"webhook_secret"`
	APIToken          string `json:"api_token"`
}

// NormalizeEnvironment keeps existing credentials on production by default.
func NormalizeEnvironment(environment string) string {
	if environment == "" {
		return EnvironmentProduction
	}
	return environment
}

func ValidEnvironment(environment string) bool {
	switch NormalizeEnvironment(environment) {
	case EnvironmentProduction, EnvironmentSandbox:
		return true
	default:
		return false
	}
}

func (c Credentials) Map() map[string]string {
	return map[string]string{
		"environment":         NormalizeEnvironment(c.Environment),
		"bank_short_name":     c.BankShortName,
		"account_number":      c.AccountNumber,
		"account_name":        c.AccountName,
		"code_prefix":         c.CodePrefix,
		"webhook_auth_method": c.WebhookAuthMethod,
		"webhook_api_key":     c.WebhookAPIKey,
		"webhook_secret":      c.WebhookSecret,
		"api_token":           c.APIToken,
	}
}

func CredentialsFromMap(values map[string]string) Credentials {
	return Credentials{
		Environment:       values["environment"],
		BankShortName:     values["bank_short_name"],
		AccountNumber:     values["account_number"],
		AccountName:       values["account_name"],
		CodePrefix:        values["code_prefix"],
		WebhookAuthMethod: values["webhook_auth_method"],
		WebhookAPIKey:     values["webhook_api_key"],
		WebhookSecret:     values["webhook_secret"],
		APIToken:          values["api_token"],
	}
}

// MergeCredentials keeps stored values for every field the update leaves empty, so an edit
// form does not force users to re-enter the account number or secrets.
func MergeCredentials(update, existing Credentials) Credentials {
	keep := func(value, stored string) string {
		if value == "" {
			return stored
		}
		return value
	}
	return Credentials{
		Environment:       keep(update.Environment, existing.Environment),
		BankShortName:     keep(update.BankShortName, existing.BankShortName),
		AccountNumber:     keep(update.AccountNumber, existing.AccountNumber),
		AccountName:       keep(update.AccountName, existing.AccountName),
		CodePrefix:        keep(update.CodePrefix, existing.CodePrefix),
		WebhookAuthMethod: keep(update.WebhookAuthMethod, existing.WebhookAuthMethod),
		WebhookAPIKey:     keep(update.WebhookAPIKey, existing.WebhookAPIKey),
		WebhookSecret:     keep(update.WebhookSecret, existing.WebhookSecret),
		APIToken:          keep(update.APIToken, existing.APIToken),
	}
}

func (c Credentials) Validate() error {
	if c.BankShortName == "" || c.AccountNumber == "" || c.AccountName == "" || c.CodePrefix == "" {
		return fmt.Errorf("%w: bank, account number, account name and code prefix are required", ErrMissingCredentials)
	}
	if !ValidEnvironment(c.Environment) {
		return fmt.Errorf("unsupported sepay environment: %s", c.Environment)
	}
	switch c.WebhookAuthMethod {
	case AuthMethodHMAC:
		if c.WebhookSecret == "" {
			return fmt.Errorf("%w: webhook secret is required for hmac auth", ErrMissingCredentials)
		}
	case AuthMethodAPIKey:
		if c.WebhookAPIKey == "" {
			return fmt.Errorf("%w: webhook api key is required for apikey auth", ErrMissingCredentials)
		}
	case AuthMethodNone, "":
	default:
		return fmt.Errorf("unknown sepay webhook auth method: %s", c.WebhookAuthMethod)
	}
	return nil
}
