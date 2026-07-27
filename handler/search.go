package handler

import (
	"strconv"
	"strings"

	"andflix/go-api/crypto"

	"github.com/gin-gonic/gin"
)

func setupPCAuth(c *gin.Context) (aeskeyInternal, deviceID, token string) {
	aeskeyInternal = c.Request.Header.Get("X-Aeskey-Internal")
	deviceID = c.Request.Header.Get("X-Device-Id")
	token = c.Request.Header.Get("X-Token")

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
	return
}

func Search(c *gin.Context) {
	var req struct {
		Size    int    `json:"size"`
		Keyword string `json:"keyword"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	aeskeyInternal, deviceID, token := setupPCAuth(c)

	body := map[string]interface{}{
		"size":          req.Size,
		"searchKeyWord": req.Keyword,
	}
	_, result, err := callPCAPI("POST", "/cms/pc/search/searchWithKeyWord", nil, body, aeskeyInternal, deviceID, token)

	response := gin.H{}
	if err != nil {
		response["error"] = err.Error()
	} else {
		if rawList, ok := result["data"].([]interface{}); ok {
			transformed := make([]gin.H, 0, len(rawList))
			for _, item := range rawList {
				m, ok := item.(map[string]interface{})
				if !ok {
					continue
				}

				if tags, ok := m["categoryTag"].([]interface{}); ok {
					skip := false
					for _, t := range tags {
						tm, ok := t.(map[string]interface{})
						if !ok {
							continue
						}
						if name, _ := tm["name"].(string); name == "LGBTQ" {
							skip = true
							break
						}
					}
					if skip {
						continue
					}
				}

				var genres []string
				if tags, ok := m["categoryTag"].([]interface{}); ok {
					for _, t := range tags {
						if tm, ok := t.(map[string]interface{}); ok {
							if name, ok := tm["name"].(string); ok {
								genres = append(genres, name)
							}
						}
					}
				}

				var countries []string
				if areas, ok := m["areas"].([]interface{}); ok {
					for _, a := range areas {
						if am, ok := a.(map[string]interface{}); ok {
							if name, ok := am["name"].(string); ok {
								countries = append(countries, name)
							}
						}
					}
				}

				year := 0
				if v, ok := m["releaseTime"].(string); ok {
					year, _ = strconv.Atoi(v)
				}

				transformed = append(transformed, gin.H{
					"id":           m["id"],
					"title":        m["name"],
					"contentType":  m["subType"],
					"genres":       genres,
					"countries":    countries,
					"year":         year,
					"rating":       m["score"],
					"cover":        imageURL(m["coverHorizontalUrl"]),
					"poster":       imageURL(m["coverVerticalUrl"]),
					"description":  m["introduction"],
					"episodeCount": m["resourceNum"],
					"status":       mapStatus(m["resourceStatus"]),
					"statusCode":   m["resourceStatus"],
				})
			}
			response["result"] = transformed
		} else {
			response["result"] = result
		}
	}

	c.Header("X-Device-Id", deviceID)
	c.Header("X-Aeskey-Internal", aeskeyInternal)
	c.JSON(200, response)
}

func GetFilter(c *gin.Context) {
	aeskeyInternal, deviceID, token := setupPCAuth(c)

	_, result, err := callPCAPI("GET", "/cms/pc/search/screen/condition", nil, nil, aeskeyInternal, deviceID, token)

	response := gin.H{}
	if err != nil {
		response["error"] = err.Error()
	} else if data, ok := result["data"]; ok {
		response["result"] = data
	} else {
		response["result"] = result
	}

	c.Header("X-Device-Id", deviceID)
	c.Header("X-Aeskey-Internal", aeskeyInternal)
	c.JSON(200, response)
}

func GetFilterResult(c *gin.Context) {
	var req struct {
		Size                int    `json:"size"`
		Params              string `json:"params"`
		SearchScreeningName string `json:"searchScreeningName"`
		Area                string `json:"area"`
		Category            string `json:"category"`
		Year                string `json:"year"`
		Order               string `json:"order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	aeskeyInternal, deviceID, token := setupPCAuth(c)

	body := map[string]interface{}{
		"size":                req.Size,
		"params":              req.Params,
		"searchScreeningName": req.SearchScreeningName,
		"area":                req.Area,
		"category":            req.Category,
		"year":                req.Year,
		"order":               req.Order,
	}
	_, result, err := callPCAPI("POST", "/cms/pc/search/screen/list", nil, body, aeskeyInternal, deviceID, token)

	response := gin.H{}
	if err != nil {
		response["error"] = err.Error()
	} else if rawList, ok := result["data"].([]interface{}); ok {
		transformed := make([]gin.H, 0, len(rawList))
		for _, item := range rawList {
			m, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			entry := gin.H{}
			for k, v := range m {
				if k == "coverVerticalUrl" {
					entry[k] = imageURL(v)
				} else {
					entry[k] = v
				}
			}
			transformed = append(transformed, entry)
		}
		response["result"] = transformed
	} else {
		response["result"] = result
	}

	c.Header("X-Device-Id", deviceID)
	c.Header("X-Aeskey-Internal", aeskeyInternal)
	c.JSON(200, response)
}
