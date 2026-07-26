package crypto

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

const PCRsaPublicKey = `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCR0hi0Ah2EkHEL0bMsyQUPn8A1
0ZW42z9LtfSiUSMaf3lw/sfqRcMTmh4m8+sBnK2a5PWKLTG2CW/HWYydN5n1BU63
c9yYMAoUD+52usxxsMaELLOb+xEv6LRW5oquDck7ZWj0xkSfHc4UXUa1l1FwVXQS
+qFIRDJGXgPCUEGVmwIDAQAB
-----END PUBLIC KEY-----`

func PCGenerateAESKeyInternal() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func PCEncryptAESKey(aeskeyInternal string) string {
	encrypted, err := RSAEncryptPKCS1v15([]byte(aeskeyInternal), PCRsaPublicKey)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(encrypted)
}

func pcRsaPublicKeyFirst16() []byte {
	cleaned := strings.NewReplacer(
		"-----BEGIN PUBLIC KEY-----", "",
		"-----END PUBLIC KEY-----", "",
		"\n", "",
		"\r", "",
	).Replace(PCRsaPublicKey)
	cleaned = strings.TrimSpace(cleaned)
	return []byte(cleaned[:16])
}

func PCComputeUserType(deviceID, aeskeyInternal string) string {
	key := pcRsaPublicKeyFirst16()
	encrypted, err := AESECBEncrypt([]byte(deviceID), key)
	if err != nil {
		return ""
	}
	b64Temp := base64.StdEncoding.EncodeToString(encrypted)
	mac := hmac.New(sha256.New, []byte(aeskeyInternal))
	mac.Write([]byte(b64Temp))
	return hex.EncodeToString(mac.Sum(nil))
}

func pcGetTimezoneString() string {
	_, offset := time.Now().Zone()
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	hours := offset / 3600
	mins := (offset % 3600) / 60
	return fmt.Sprintf("GMT%s%02d:%02d", sign, hours, mins)
}

func wurlencode(s string) string {
	encoded := url.QueryEscape(s)
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "!", "%21")
	encoded = strings.ReplaceAll(encoded, "'", "%27")
	encoded = strings.ReplaceAll(encoded, "(", "%28")
	encoded = strings.ReplaceAll(encoded, ")", "%29")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	return encoded
}

func objectToSignString(obj map[string]interface{}) string {
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		v := obj[k]
		switch val := v.(type) {
		case string:
			if val != "" {
				parts = append(parts, wurlencode(val))
			} else {
				parts = append(parts, "")
			}
		case []interface{}:
			for _, item := range val {
				switch itemVal := item.(type) {
				case string:
					parts = append(parts, itemVal)
				case map[string]interface{}:
					parts = append(parts, objectToSignString(itemVal))
				default:
					b, _ := json.Marshal(itemVal)
					parts = append(parts, string(b))
				}
			}
		case map[string]interface{}:
			parts = append(parts, objectToSignString(val))
		default:
			b, _ := json.Marshal(v)
			parts = append(parts, string(b))
		}
	}
	return strings.Join(parts, "")
}

func PCBuildSignString(query map[string]string, body interface{}) string {
	if len(query) > 0 {
		keys := make([]string, 0, len(query))
		for k := range query {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var values []string
		for _, k := range keys {
			v := query[k]
			if v != "" {
				values = append(values, wurlencode(v))
			} else {
				values = append(values, "")
			}
		}
		return strings.Join(values, "")
	}

	if body == nil {
		return ""
	}

	if bodyStr, ok := body.(string); ok {
		var parsed interface{}
		if err := json.Unmarshal([]byte(bodyStr), &parsed); err == nil {
			body = parsed
		} else {
			return bodyStr
		}
	}

	switch val := body.(type) {
	case map[string]interface{}:
		return objectToSignString(val)
	case []interface{}:
		if len(val) > 0 {
			if first, ok := val[0].(map[string]interface{}); ok {
				return objectToSignString(first)
			}
		}
		var items []string
		for _, item := range val {
			switch itemVal := item.(type) {
			case string:
				items = append(items, itemVal)
			default:
				b, _ := json.Marshal(itemVal)
				items = append(items, string(b))
			}
		}
		return strings.Join(items, ",")
	default:
		return ""
	}
}

func PCComputeSign(aeskeyInternal, currentTimeMs string, query map[string]string, body interface{}) string {
	s := PCBuildSignString(query, body)
	data := []byte(s + currentTimeMs)
	encrypted, err := AESECBEncrypt(data, []byte(aeskeyInternal))
	if err != nil {
		return ""
	}
	b64Result := base64.StdEncoding.EncodeToString(encrypted)
	hash := md5.Sum([]byte(b64Result))
	return hex.EncodeToString(hash[:])
}

func PCDecryptResponse(encryptedDataB64, aeskeyInternal string) (interface{}, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedDataB64)
	if err != nil {
		return nil, err
	}
	decrypted, err := AESECBDecrypt(ciphertext, []byte(aeskeyInternal))
	if err != nil {
		return nil, err
	}
	var result interface{}
	if err := json.Unmarshal(decrypted, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func PCBuildHeaders(aeskeyInternal, deviceID string, query map[string]string, body interface{}, token string) map[string]string {
	currentTime := fmt.Sprintf("%d", time.Now().UnixMilli())
	aeskey := PCEncryptAESKey(aeskeyInternal)
	sign := PCComputeSign(aeskeyInternal, currentTime, query, body)
	userType := PCComputeUserType(deviceID, aeskeyInternal)

	headers := map[string]string{
		"aeskey":          aeskey,
		"aeskey_internal": aeskeyInternal,
		"clienttype":      "web-loklok",
		"currenttime":     currentTime,
		"deviceid":        deviceID,
		"keke":            "false",
		"lang":            "in_ID",
		"sign":            sign,
		"timezone":        pcGetTimezoneString(),
		"usertype":        userType,
		"versioncode":     "32",
		"content-type":    "application/json",
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36",
		"Origin":          "https://www.loklok.fun",
	}
	if token != "" {
		headers["token"] = token
	}
	return headers
}
