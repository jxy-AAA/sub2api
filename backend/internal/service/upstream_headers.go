package service

import (
	"net/http"
	"strings"
)

func applyAccountCredentialHeaders(headers http.Header, account *Account) {
	if headers == nil || account == nil {
		return
	}
	for key, value := range account.GetCredentialHeaders() {
		trimmedKey := strings.TrimSpace(key)
		trimmedValue := strings.TrimSpace(value)
		if trimmedKey == "" || trimmedValue == "" {
			continue
		}
		if wireKey, ok := headerWireCasing[strings.ToLower(trimmedKey)]; ok {
			setHeaderRaw(headers, wireKey, trimmedValue)
			continue
		}
		headers.Set(http.CanonicalHeaderKey(trimmedKey), trimmedValue)
	}
}
