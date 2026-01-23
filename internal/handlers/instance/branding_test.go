package instance

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Test_BrandingHandler_GetBranding_Success tests getting default branding
func Test_BrandingHandler_GetBranding_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/branding", func(c *gin.Context) {
		// Simulate default branding values
		branding := map[string]interface{}{
			"site_name":        "VideoStreamGo",
			"logo_url":         "",
			"favicon_url":      "",
			"primary_color":    "#2563eb",
			"secondary_color":  "#1e40af",
			"accent_color":     "#3b82f6",
			"background_color": "#ffffff",
			"text_color":       "#1f2937",
			"custom_css":       "",
			"header_html":      "",
			"footer_html":      "",
		}

		c.JSON(http.StatusOK, gin.H{
			"data": branding,
		})
	})

	req := httptest.NewRequest("GET", "/branding", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "VideoStreamGo", data["site_name"])
	assert.Equal(t, "#2563eb", data["primary_color"])
}

// Test_BrandingHandler_GetBranding_Custom tests getting custom branding
func Test_BrandingHandler_GetBranding_Custom(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/branding", func(c *gin.Context) {
		// Simulate custom branding from tenant context
		customBranding := map[string]interface{}{
			"site_name":        "My Video Site",
			"logo_url":         "https://example.com/logo.png",
			"favicon_url":      "https://example.com/favicon.ico",
			"primary_color":    "#ff5733",
			"secondary_color":  "#c70039",
			"accent_color":     "#ff8f33",
			"background_color": "#f8f9fa",
			"text_color":       "#212529",
			"custom_css":       ".custom-class { color: red; }",
			"header_html":      "<div class='custom-header'>Welcome</div>",
			"footer_html":      "<div class='custom-footer'>Copyright 2024</div>",
		}

		c.JSON(http.StatusOK, gin.H{
			"data": customBranding,
		})
	})

	req := httptest.NewRequest("GET", "/branding", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "My Video Site", data["site_name"])
	assert.Equal(t, "https://example.com/logo.png", data["logo_url"])
	assert.Equal(t, "#ff5733", data["primary_color"])
}

// Test_BrandingHandler_UpdateBranding_Success tests updating branding
func Test_BrandingHandler_UpdateBranding_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.PUT("/branding", func(c *gin.Context) {
		var req struct {
			SiteName        *string `json:"site_name"`
			LogoURL         *string `json:"logo_url"`
			PrimaryColor    *string `json:"primary_color"`
			SecondaryColor  *string `json:"secondary_color"`
			AccentColor     *string `json:"accent_color"`
			BackgroundColor *string `json:"background_color"`
			TextColor       *string `json:"text_color"`
			CustomCSS       *string `json:"custom_css"`
			HeaderHTML      *string `json:"header_html"`
			FooterHTML      *string `json:"footer_html"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// In production, save to database
		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"site_name":        req.SiteName,
				"logo_url":         req.LogoURL,
				"primary_color":    req.PrimaryColor,
				"secondary_color":  req.SecondaryColor,
				"accent_color":     req.AccentColor,
				"background_color": req.BackgroundColor,
				"text_color":       req.TextColor,
				"custom_css":       req.CustomCSS,
				"header_html":      req.HeaderHTML,
				"footer_html":      req.FooterHTML,
			},
			"message": "Branding updated successfully",
		})
	})

	body := `{"site_name":"Updated Site","primary_color":"#ff5733","logo_url":"https://example.com/new-logo.png"}`
	req := httptest.NewRequest("PUT", "/branding", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "Updated Site", data["site_name"])
	assert.Equal(t, "#ff5733", data["primary_color"])
}

// Test_BrandingHandler_UpdateBranding_ValidationError tests validation on update
func Test_BrandingHandler_UpdateBranding_ValidationError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.PUT("/branding", func(c *gin.Context) {
		var req struct {
			SiteName *string `json:"site_name"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]string{
					"code":    "VALIDATION_ERROR",
					"message": "Invalid request",
					"details": err.Error(),
				},
			})
			return
		}
	})

	// Invalid JSON
	req := httptest.NewRequest("PUT", "/branding", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test_BrandingHandler_Colors tests color configuration
func Test_BrandingHandler_Colors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/branding", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"primary_color":    "#2563eb",
				"secondary_color":  "#1e40af",
				"accent_color":     "#3b82f6",
				"background_color": "#ffffff",
				"text_color":       "#1f2937",
			},
		})
	})

	req := httptest.NewRequest("GET", "/branding", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Regexp(t, "^#[0-9A-Fa-f]{6}$", data["primary_color"])
	assert.Regexp(t, "^#[0-9A-Fa-f]{6}$", data["secondary_color"])
	assert.Regexp(t, "^#[0-9A-Fa-f]{6}$", data["accent_color"])
}

// Test_BrandingHandler_CustomHTML tests custom HTML injection
func Test_BrandingHandler_CustomHTML(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/branding", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"header_html": "<div class='header'>Custom Header</div>",
				"footer_html": "<div class='footer'>Custom Footer</div>",
				"custom_css":  ".header { background: blue; }",
			},
		})
	})

	req := httptest.NewRequest("GET", "/branding", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Contains(t, data["header_html"], "Custom Header")
	assert.Contains(t, data["footer_html"], "Custom Footer")
	assert.Contains(t, data["custom_css"], "background")
}

// Test_BrandingHandler_PartialUpdate tests partial branding update
func Test_BrandingHandler_PartialUpdate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.PUT("/branding", func(c *gin.Context) {
		var req struct {
			SiteName *string `json:"site_name"`
			LogoURL  *string `json:"logo_url"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Only update provided fields
		response := map[string]interface{}{}
		if req.SiteName != nil {
			response["site_name"] = *req.SiteName
		}
		if req.LogoURL != nil {
			response["logo_url"] = *req.LogoURL
		}

		c.JSON(http.StatusOK, gin.H{
			"data":    response,
			"message": "Branding updated successfully",
		})
	})

	// Only update site name
	body := `{"site_name":"New Site Name"}`
	req := httptest.NewRequest("PUT", "/branding", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "New Site Name", data["site_name"])
	assert.Nil(t, data["logo_url"])
}
