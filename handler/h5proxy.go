package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"andflix/go-api/crypto"

	"github.com/gin-gonic/gin"
)

func proxyH5(c *gin.Context, endpointPath string) {
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

	deviceID := c.Request.Header.Get("X-Device-Id")
	token := c.Request.Header.Get("X-Token")

	if deviceID == "" && token != "" {
		deviceID = extractDeviceIDFromToken(token)
	}

	if deviceID == "" {
		if cached, ok := deviceKeyCache.Load("h5_last_device"); ok {
			deviceID = cached.(string)
		}
	}
	if deviceID == "" {
		deviceID = crypto.GenerateDeviceID()
	}
	deviceKeyCache.Store("h5_last_device", deviceID)

	var params map[string]interface{}
	if method == "GET" {
		params = crypto.H5ParamsFromQuery(query)
	} else {
		params = crypto.H5ParamsFromBody(body)
	}

	headers := crypto.H5BuildSignHeaders(params, deviceID, token)

	reqURL := h5APIBase + endpointPath
	if len(query) > 0 && method == "GET" {
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
		} else {
			reqBody = bytes.NewReader(bodyBytes)
		}
	}

	upstreamResp, err := forwardRequest(method, reqURL, headers, reqBody)
	if err != nil {
		c.JSON(502, gin.H{"error": "H5 proxy request failed: " + err.Error()})
		return
	}
	defer upstreamResp.Body.Close()

	respBody, _ := io.ReadAll(upstreamResp.Body)

	var respJSON interface{}
	if err := json.Unmarshal(respBody, &respJSON); err != nil {
		c.Data(upstreamResp.StatusCode, upstreamResp.Header.Get("Content-Type"), respBody)
		return
	}

	c.Header("X-Device-Id", deviceID)
	c.JSON(upstreamResp.StatusCode, respJSON)
}

func H5MovieDramaGet(c *gin.Context) {
	proxyH5(c, "/cms/web/movieDrama/get")
}

func H5GetPlayInfo(c *gin.Context) {
	proxyH5(c, "/cms/web/ios_h5/movieDrama/getPlayInfo")
}
