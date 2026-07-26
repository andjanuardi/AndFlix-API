package crypto

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"
)

const H5PublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQC7GW1zgx9/ssgCjoZhuCvISy5N
s9T2UgAzjJqS2uTGuCVtZsN3TE5wd4OIeiVG2TVDH2Gxlzrxd5jg7P6IiUKqsliS
dZxx/ceqLDawKgvO8mJ+hJJsuIxSL7Bi6T0p+xH6ibw4orGfCFUJhGryE9hqp9qT
RiHOMvgC2si1VqrgaQIDAQAB
-----END PUBLIC KEY-----`

const h5Alphanum = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func H5GenerateRandomKey(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = h5Alphanum[rand.Intn(len(h5Alphanum))]
	}
	return string(b)
}

func h5RSAEncrypt(data string) string {
	encrypted, err := RSAEncryptPKCS1v15([]byte(data), H5PublicKeyPEM)
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(encrypted)
}

func h5AESECBEncrypt(plaintext, key string) string {
	encrypted, err := AESECBEncrypt([]byte(plaintext), []byte(key))
	if err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(encrypted)
}

func h5MD5Hex(data string) string {
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func H5ConvertObj(params map[string]interface{}, topLevel bool) string {
	type kv struct {
		key, value string
	}
	var entries []kv

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, r := range keys {
		n := params[r]
		if n == nil {
			continue
		}
		switch v := n.(type) {
		case []interface{}:
			for i, val := range v {
				switch val2 := val.(type) {
				case map[string]interface{}:
					entries = append(entries, kv{r, H5ConvertObj(val2, true)})
				case []interface{}:
					entries = append(entries, kv{r, H5ConvertObj(map[string]interface{}{"key": val2}, false)})
				default:
					entries = append(entries, kv{fmt.Sprintf("%s[%d]", r, i), fmt.Sprintf("%v", val)})
				}
			}
		case map[string]interface{}:
			entries = append(entries, kv{r, H5ConvertObj(v, true)})
		default:
			entries = append(entries, kv{r, fmt.Sprintf("%v", n)})
		}
	}

	if topLevel {
		grouped := make(map[string][]string)
		for _, e := range entries {
			grouped[e.key] = append(grouped[e.key], e.value)
		}
		sortedKeys := make([]string, 0, len(grouped))
		for k := range grouped {
			sortedKeys = append(sortedKeys, k)
		}
		sort.Strings(sortedKeys)
		var result strings.Builder
		for _, k := range sortedKeys {
			for _, v := range grouped[k] {
				result.WriteString(v)
			}
		}
		return result.String()
	}

	var result strings.Builder
	for _, e := range entries {
		result.WriteString(e.value)
	}
	return result.String()
}

func h5GetSign(params map[string]interface{}, randomKey, currentTime string) string {
	serialized := H5ConvertObj(params, true)
	encoded := base64.StdEncoding.EncodeToString([]byte(serialized))
	r := strings.ReplaceAll(currentTime+encoded, "+", "-")
	r = strings.ReplaceAll(r, "/", "_")
	encrypted := h5AESECBEncrypt(r, randomKey)
	return h5MD5Hex(encrypted)
}

func H5BuildSignHeaders(params map[string]interface{}, deviceID, token string) map[string]string {
	randomKey := H5GenerateRandomKey(16)
	currentTime := fmt.Sprintf("%d", time.Now().UnixMilli())

	if params == nil {
		params = make(map[string]interface{})
	}
	sign := h5GetSign(params, randomKey, currentTime)
	aesKey := h5RSAEncrypt(randomKey)

	headers := map[string]string{
		"currentTime":  currentTime,
		"sign":         sign,
		"aesKey":       aesKey,
		"clientType":   "ANDROID",
		"versionCode":  "42",
		"lang":         "in_ID",
		"timezone":     "GMT+07:00",
		"content-type": "application/json",
	}
	if deviceID != "" {
		headers["deviceid"] = deviceID
	}
	if token != "" {
		headers["token"] = token
	}
	return headers
}

func H5ParamsFromQuery(query map[string]string) map[string]interface{} {
	params := make(map[string]interface{}, len(query))
	for k, v := range query {
		params[k] = v
	}
	return params
}

func H5ParamsFromBody(body interface{}) map[string]interface{} {
	switch v := body.(type) {
	case map[string]interface{}:
		return v
	case string:
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(v), &parsed); err == nil {
			return parsed
		}
	}
	return nil
}
