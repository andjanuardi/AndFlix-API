package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"andflix/go-api/crypto"

	"github.com/gin-gonic/gin"
)

func callPCAPI(method, endpoint string, query map[string]string, body interface{}, aeskeyInternal, deviceID, token string) (int, map[string]interface{}, error) {
	headers := crypto.PCBuildHeaders(aeskeyInternal, deviceID, query, body, token)

	reqURL := pcAPIBase + endpoint
	if len(query) > 0 {
		params := url.Values{}
		for k, v := range query {
			params.Set(k, v)
		}
		reqURL += "?" + params.Encode()
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	resp, err := forwardRequest(method, reqURL, headers, reqBody)
	if err != nil {
		return 0, nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var respJSON map[string]interface{}
	if err := json.Unmarshal(respBody, &respJSON); err != nil {
		return resp.StatusCode, nil, fmt.Errorf("non-json response from %s: %s", endpoint, string(respBody))
	}

	if resp.Header.Get("ecy") == "1" {
		if dataStr, ok := respJSON["data"].(string); ok && dataStr != "" {
			if decrypted, err := crypto.PCDecryptResponse(dataStr, aeskeyInternal); err == nil {
				respJSON["data"] = decrypted
			}
		}
	}

	return resp.StatusCode, respJSON, nil
}

func proxyPC(c *gin.Context, endpointPath string) {
	method := c.Request.Method

	query := make(map[string]string)
	for k, v := range c.Request.URL.Query() {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}

	var body interface{}
	bodyBytes, _ := io.ReadAll(c.Request.Body)
	if len(bodyBytes) > 0 {
		contentType := c.Request.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			json.Unmarshal(bodyBytes, &body)
		} else {
			body = string(bodyBytes)
		}
	}

	aeskeyInternal := c.Request.Header.Get("X-Aeskey-Internal")
	deviceID := c.Request.Header.Get("X-Device-Id")
	token := c.Request.Header.Get("X-Token")

	if deviceID == "" && token != "" {
		deviceID = extractDeviceIDFromToken(token)
	}

	if deviceID != "" && aeskeyInternal == "" {
		if cached, ok := deviceKeyCache.Load(deviceID); ok {
			aeskeyInternal = cached.(string)
		}
	}

	if aeskeyInternal == "" {
		aeskeyInternal = crypto.PCGenerateAESKeyInternal()
	} else {
		aeskeyInternal = strings.TrimSpace(aeskeyInternal)
	}

	if deviceID == "" {
		deviceID = crypto.GenerateDeviceID()
	} else {
		deviceID = strings.TrimSpace(deviceID)
	}

	if deviceID != "" {
		deviceKeyCache.Store(deviceID, aeskeyInternal)
	}

	headers := crypto.PCBuildHeaders(aeskeyInternal, deviceID, query, body, token)

	reqURL := pcAPIBase + endpointPath
	if len(query) > 0 {
		params := c.Request.URL.Query()
		reqURL += "?" + params.Encode()
	}

	var reqBody io.Reader
	if bodyBytes != nil && method != "GET" {
		if bodyMap, ok := body.(map[string]interface{}); ok {
			b, _ := json.Marshal(bodyMap)
			reqBody = bytes.NewReader(b)
		} else if bodyList, ok := body.([]interface{}); ok {
			b, _ := json.Marshal(bodyList)
			reqBody = bytes.NewReader(b)
		} else if bodyStr, ok := body.(string); ok {
			reqBody = strings.NewReader(bodyStr)
		} else {
			reqBody = bytes.NewReader(bodyBytes)
		}
	}

	upstreamResp, err := forwardRequest(method, reqURL, headers, reqBody)
	if err != nil {
		c.JSON(502, gin.H{"error": "Proxy request failed: " + err.Error()})
		return
	}
	defer upstreamResp.Body.Close()

	respBody, _ := io.ReadAll(upstreamResp.Body)

	var respJSON map[string]interface{}
	if err := json.Unmarshal(respBody, &respJSON); err != nil {
		for k, v := range upstreamResp.Header {
			if len(v) > 0 {
				c.Header(k, v[0])
			}
		}
		c.Data(upstreamResp.StatusCode, upstreamResp.Header.Get("Content-Type"), respBody)
		return
	}

	ecy := upstreamResp.Header.Get("ecy")
	if ecy == "1" {
		if dataStr, ok := respJSON["data"].(string); ok && dataStr != "" {
			decrypted, err := crypto.PCDecryptResponse(dataStr, aeskeyInternal)
			if err == nil {
				respJSON["data"] = decrypted
			}
		}
	}

	forwardHeaders := []string{"ecy", "lc", "set-cookie", "currentTime", "versionStatus", "UM_Event_Country", "Content-Type"}
	for _, h := range forwardHeaders {
		if v := upstreamResp.Header.Get(h); v != "" {
			c.Header(h, v)
		}
	}

	c.Header("X-Device-Id", deviceID)
	c.Header("X-Aeskey-Internal", aeskeyInternal)
	c.JSON(upstreamResp.StatusCode, respJSON)
}

func PCSearch(c *gin.Context) {
	proxyPC(c, "/cms/pc/search/searchWithKeyWord")
}

func imageURL(raw interface{}) string {
	s, ok := raw.(string)
	if !ok || s == "" {
		return ""
	}
	return "/image?url=" + url.QueryEscape(s)
}

func mapStatus(v interface{}) string {
	switch v := v.(type) {
	case float64:
		switch int(v) {
		case 1:
			return "Sedang Tayang"
		case 2:
			return "Selesai"
		}
	}
	return ""
}

func GetHomeCombined(c *gin.Context) {
	var req struct {
		Page  int `json:"page"`
		NavID int `json:"navId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	aeskeyInternal := c.Request.Header.Get("X-Aeskey-Internal")
	deviceID := c.Request.Header.Get("X-Device-Id")
	token := c.Request.Header.Get("X-Token")

	if deviceID == "" && token != "" {
		deviceID = extractDeviceIDFromToken(token)
	}
	if deviceID != "" && aeskeyInternal == "" {
		if cached, ok := deviceKeyCache.Load(deviceID); ok {
			aeskeyInternal = cached.(string)
		}
	}
	if aeskeyInternal == "" {
		aeskeyInternal = crypto.PCGenerateAESKeyInternal()
	} else {
		aeskeyInternal = strings.TrimSpace(aeskeyInternal)
	}
	if deviceID == "" {
		deviceID = crypto.GenerateDeviceID()
	} else {
		deviceID = strings.TrimSpace(deviceID)
	}
	if deviceID != "" {
		deviceKeyCache.Store(deviceID, aeskeyInternal)
	}

	navCode, navResult, navErr := callPCAPI("GET", "/home/pc/navigationBar", nil, nil, aeskeyInternal, deviceID, token)
	_ = navCode

	query := map[string]string{
		"page":         strconv.Itoa(req.Page),
		"navigationId": strconv.Itoa(req.NavID),
	}
	homeCode, homeResult, homeErr := callPCAPI("GET", "/home/pc/getHome", query, nil, aeskeyInternal, deviceID, token)
	_ = homeCode

	response := gin.H{}
	if navErr != nil {
		response["navList"] = gin.H{"error": navErr.Error()}
	} else if navData, ok := navResult["data"].(map[string]interface{}); ok {
		items, _ := navData["navigationBarItemList"].([]interface{})
		simpleNav := make([]gin.H, 0, len(items))
		for _, item := range items {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			if name, _ := m["name"].(string); name == "Peringkat" {
				continue
			}
			simpleNav = append(simpleNav, gin.H{
				"id":   m["id"],
				"name": m["name"],
			})
		}
		response["navList"] = simpleNav
	} else {
		response["navList"] = navResult
	}
	if homeErr != nil {
		response["error"] = homeErr.Error()
	} else if homeData, ok := homeResult["data"].(map[string]interface{}); ok {
		if rawList, ok := homeData["recommendItems"].([]interface{}); ok {
			transformed := make([]gin.H, 0, len(rawList))
			for _, item := range rawList {
				m, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				entry := gin.H{}
				if v, ok := m["homeSectionType"]; ok {
					entry["type"] = v
				}
				if v, ok := m["homeSectionName"]; ok {
					entry["Name"] = v
				}
				if rawItems, ok := m["recommendContentVOList"].([]interface{}); ok {
					contentList := make([]gin.H, 0, len(rawItems))
					for _, ci := range rawItems {
						cm, ok := ci.(map[string]interface{})
						if !ok {
							continue
						}
						if tags, ok := cm["tagList"].([]interface{}); ok {
							skip := false
							for _, t := range tags {
								if s, ok := t.(string); ok && s == "LGBTQ" {
									skip = true
									break
								}
							}
							if skip {
								continue
							}
						}
						contentList = append(contentList, gin.H{
							"id":           cm["id"],
							"title":        cm["title"],
							"contentType":  cm["contentType"],
							"genres":       cm["tagList"],
							"year":         cm["releaseTime"],
							"rating":       cm["score"],
							"cover":        imageURL(cm["coverHorizontalUrl"]),
							"poster":       imageURL(cm["imageUrl"]),
							"description":  cm["introduction"],
							"episodeCount": cm["resourceNum"],
							"status":       mapStatus(cm["resourceStatus"]),
							"statusCode":   cm["resourceStatus"],
						})
					}
					entry["ContentList"] = contentList
				}
				transformed = append(transformed, entry)
			}
			response["sectionList"] = transformed
		}
	} else {
		response["sectionList"] = homeResult
	}

	c.Header("X-Device-Id", deviceID)
	c.Header("X-Aeskey-Internal", aeskeyInternal)
	c.JSON(200, response)
}
