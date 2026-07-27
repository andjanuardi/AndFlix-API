package handler

import (
	"fmt"
	"strconv"
	"strings"

	"andflix/go-api/crypto"

	"github.com/gin-gonic/gin"
)

func transformContentListItem(cm map[string]interface{}) gin.H {
	return gin.H{
		"id":           cm["id"],
		"title":        cm["name"],
		"contentType":  cm["subType"],
		"genres":       extractNames(cm["tagList"]),
		"countries":    cm["areaNameList"],
		"year":         cm["year"],
		"rating":       cm["score"],
		"cover":        imageURL(cm["coverHorizontalUrl"]),
		"poster":       imageURL(cm["coverVerticalUrl"]),
		"description":  cm["introduction"],
		"episodeCount": cm["resourceNum"],
		"status":       mapStatus(cm["resourceStatus"]),
		"statusCode":   cm["resourceStatus"],
	}
}

func extractNames(raw interface{}) []string {
	list, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	var names []string
	for _, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if name, ok := m["name"].(string); ok {
			names = append(names, name)
		}
	}
	return names
}

func filterSubtitles(raw interface{}) interface{} {
	list, ok := raw.([]interface{})
	if !ok {
		return raw
	}
	filtered := make([]interface{}, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		abbr, _ := m["languageAbbr"].(string)
		if strings.HasPrefix(abbr, "in") || strings.HasPrefix(abbr, "en") {
			m["subtitlingUrl"] = subtitleURL(m["subtitlingUrl"])
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func fmtID(raw interface{}) string {
	switch v := raw.(type) {
	case float64:
		return strconv.Itoa(int(v))
	case int:
		return strconv.Itoa(v)
	default:
		return fmt.Sprint(raw)
	}
}

func hasLGBTQ(raw interface{}) bool {
	tags, ok := raw.([]interface{})
	if !ok {
		return false
	}
	for _, t := range tags {
		m, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		if name, _ := m["name"].(string); name == "LGBTQ" {
			return true
		}
	}
	return false
}

func GetDetail(c *gin.Context) {
	var req struct {
		ID       int `json:"id"`
		Category int `json:"category"`
		Episode  int `json:"episode"`
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

	idStr := strconv.Itoa(req.ID)
	detailQuery := map[string]string{
		"id":       idStr,
		"category": strconv.Itoa(req.Category),
	}

	_, detailResult, detailErr := callH5API("/cms/web/movieDrama/get", detailQuery, aeskeyInternal, deviceID, token)

	response := gin.H{}
	if detailErr != nil {
		c.JSON(502, gin.H{"error": "detail request failed: " + detailErr.Error()})
		return
	}

	if detailData, ok := detailResult["data"].(map[string]interface{}); ok {
		response["id"] = detailData["id"]
		response["title"] = detailData["name"]
		if dt, ok := detailData["drameTypeVo"].(map[string]interface{}); ok {
			response["type"] = dt["drameName"]
			response["contentType"] = dt["drameType"]
		}
		response["category"] = detailData["category"]
		response["genres"] = extractNames(detailData["tagList"])
		response["countries"] = detailData["areaNameList"]
		response["year"] = detailData["year"]
		response["rating"] = detailData["score"]
		response["cover"] = imageURL(detailData["coverHorizontalUrl"])
		response["poster"] = imageURL(detailData["coverVerticalUrl"])
		response["description"] = detailData["introduction"]
		response["episodeCount"] = detailData["episodeCount"]
		if eps, ok := detailData["episodeVo"].([]interface{}); ok {
			response["episodeRelease"] = len(eps)
		}
		response["status"] = mapStatus(detailData["resourceStatus"])
		response["statusCode"] = detailData["resourceStatus"]

		// Recommendations (likeList) — transform seperti ContentList
		if likes, ok := detailData["likeList"].([]interface{}); ok {
			recList := make([]gin.H, 0, len(likes))
			for _, li := range likes {
				lm, ok := li.(map[string]interface{})
				if !ok {
					continue
				}
				if hasLGBTQ(lm["tagList"]) {
					continue
				}
				recList = append(recList, transformContentListItem(lm))
			}
			response["recommendations"] = recList
		}

		// Episode & PlayInfo (hanya jika episode > 0)
		if req.Episode > 0 {
			var episodeID string
			var matchedEp gin.H
			var defList []interface{}
			var fallbackDef, fallbackURL string
			hasEpisodes := false

			// TV/Series: cari episode dari episodeVo
			if eps, ok := detailData["episodeVo"].([]interface{}); ok && len(eps) > 0 {
				hasEpisodes = true
				for _, ep := range eps {
					em, ok := ep.(map[string]interface{})
					if !ok {
						continue
					}
					var sn int
					switch v := em["seriesNo"].(type) {
					case float64:
						sn = int(v)
					case int:
						sn = v
					}
					if sn == req.Episode {
						episodeID = fmtID(em["id"])
						defList, _ = em["definitionList"].([]interface{})

						matchedEp = gin.H{
							"id":        em["id"],
							"seriesNo":  em["seriesNo"],
							"name":      em["name"],
							"subtitles": filterSubtitles(em["subtitlingList"]),
							"duration":  em["totalTime"],
						}
						break
					}
				}
				// Fallback untuk MOVIE: gunakan episode pertama jika tidak ada match
				if episodeID == "" && matchedEp == nil {
					if em, ok := eps[0].(map[string]interface{}); ok {
						episodeID = fmtID(em["id"])
						defList, _ = em["definitionList"].([]interface{})

						matchedEp = gin.H{
							"id":        em["id"],
							"seriesNo":  em["seriesNo"],
							"name":      em["name"],
							"subtitles": filterSubtitles(em["subtitlingList"]),
							"duration":  em["totalTime"],
						}
					}
				}
			}

			// Call playInfo
			playQuery := map[string]string{
				"id":       idStr,
				"category": strconv.Itoa(req.Category),
			}
			if episodeID != "" {
				playQuery["episodeId"] = episodeID
			}
			_, playResult, playErr := callH5API("/cms/web/ios_h5/movieDrama/getPlayInfo", playQuery, aeskeyInternal, deviceID, token)

			if playErr == nil {
				if playData, ok := playResult["data"].(map[string]interface{}); ok {
					if !hasEpisodes {
						defList, _ = playData["definitionList"].([]interface{})
						matchedEp = gin.H{
							"id":        detailData["id"],
							"seriesNo":  1,
							"name":      detailData["name"],
							"subtitles": filterSubtitles(playData["subtitlingList"]),
							"duration":  playData["totalDuration"],
						}
					}
					response["totalDuration"] = playData["totalDuration"]
					fallbackDef, _ = playData["currentDefinition"].(string)
					fallbackURL, _ = playData["mediaUrl"].(string)
				}
			}
			if _, ok := response["totalDuration"]; !ok {
				response["totalDuration"] = nil
			}

			// Build urls dari defList
			if matchedEp != nil {
				var urls []gin.H
				for _, d := range defList {
					dm, ok := d.(map[string]interface{})
					if !ok {
						continue
					}
					code, _ := dm["code"].(string)
					if code == "" {
						continue
					}
					defQuery := map[string]string{
						"id":         idStr,
						"category":   strconv.Itoa(req.Category),
						"definition": code,
					}
					if episodeID != "" {
						defQuery["episodeId"] = episodeID
					}
					_, defResult, defErr := callH5API("/cms/web/ios_h5/movieDrama/getPlayInfo", defQuery, aeskeyInternal, deviceID, token)
					if defErr == nil {
						if defData, ok := defResult["data"].(map[string]interface{}); ok {
							if u, ok := defData["mediaUrl"].(string); ok && u != "" {
								urls = append(urls, gin.H{
									"definition": code,
									"url":        streamURL(u),
								})
							}
						}
					}
				}
				if len(urls) == 0 && fallbackDef != "" && fallbackURL != "" {
					urls = []gin.H{{"definition": fallbackDef, "url": streamURL(fallbackURL)}}
				}
				if urls == nil {
					urls = []gin.H{}
				}
				matchedEp["urls"] = urls
				response["playerInfo"] = matchedEp
			}
		}
	} else {
		response["error"] = "no data in detail response"
	}

	c.Header("X-Device-Id", deviceID)
	c.Header("X-Aeskey-Internal", aeskeyInternal)
	c.JSON(200, response)
}
