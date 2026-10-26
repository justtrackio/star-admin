package internal

import "testing"

func TestFrontendConfigRedaction(t *testing.T) {
	handler := &HandlerOverview{}
	for _, key := range []string{"authentication_ldap_simple_bind_root_pwd", "jdbc_password", "AWS_ACCESS_KEY", "cloud_secret", "auth_token", "private_key", "keytab_path"} {
		setting := map[string]string{"Key": key, "Value": "test-credential", "Type": "String", "Comment": "description"}
		handler.redactFrontendConfig(setting)

		if setting["Value"] != "******" || setting["Comment"] != "description" || setting["Type"] != "String" {
			t.Fatalf("credential not masked or metadata changed for %s", key)
		}
	}

	for _, value := range []string{"", "false", "0", "[]", "9030"} {
		setting := map[string]string{"Key": "ordinary_setting", "Value": value}
		handler.redactFrontendConfig(setting)

		if setting["Value"] != value {
			t.Fatalf("ordinary setting changed for %q", value)
		}
	}

	alias := map[string]string{"Key": "renamed_setting", "AliasNames": "[old_password]", "Value": "test-credential"}
	handler.redactFrontendConfig(alias)

	if alias["Value"] != "******" {
		t.Fatal("credential alias not masked")
	}

	empty := map[string]string{"Key": "password", "Value": ""}
	handler.redactFrontendConfig(empty)

	if empty["Value"] != "" {
		t.Fatal("unset credential must remain empty")
	}
}

func TestFrontendConfigKeepsNonSecretTokenSettings(t *testing.T) {
	handler := &HandlerOverview{}
	setting := map[string]string{"Key": "arrow_token_cache_size", "Type": "int", "Value": "4096"}
	handler.redactFrontendConfig(setting)

	if setting["Value"] != "4096" {
		t.Fatal("numeric token cache settings are not credentials")
	}
}
