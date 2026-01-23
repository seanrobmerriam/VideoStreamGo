package instance

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"videostreamgo/internal/middleware"
	"videostreamgo/internal/types"
)

// BrandingHandler handles branding endpoints for instance API
type BrandingHandler struct {
	// In production, this would fetch branding from the instance database or config
}

// NewBrandingHandler creates a new BrandingHandler
func NewBrandingHandler() *BrandingHandler {
	return &BrandingHandler{}
}

// GetBranding returns the branding configuration for the instance
func (h *BrandingHandler) GetBranding(c *gin.Context) {
	tenantCtx := middleware.GetTenantContext(c)

	// Default branding values
	siteName := "VideoStreamGo"
	logoURL := ""
	faviconURL := ""
	primaryColor := "#2563eb"
	secondaryColor := "#1e40af"
	accentColor := "#3b82f6"
	backgroundColor := "#ffffff"
	textColor := "#1f2937"
	customCSS := ""
	headerHTML := ""
	footerHTML := ""

	if tenantCtx != nil && tenantCtx.BrandingConfig != nil {
		if name, ok := tenantCtx.BrandingConfig["site_name"]; ok {
			siteName = name
		}
		if url, ok := tenantCtx.BrandingConfig["logo_url"]; ok {
			logoURL = url
		}
		if url, ok := tenantCtx.BrandingConfig["favicon_url"]; ok {
			faviconURL = url
		}
		if color, ok := tenantCtx.BrandingConfig["primary_color"]; ok {
			primaryColor = color
		}
		if color, ok := tenantCtx.BrandingConfig["secondary_color"]; ok {
			secondaryColor = color
		}
		if color, ok := tenantCtx.BrandingConfig["accent_color"]; ok {
			accentColor = color
		}
		if color, ok := tenantCtx.BrandingConfig["background_color"]; ok {
			backgroundColor = color
		}
		if color, ok := tenantCtx.BrandingConfig["text_color"]; ok {
			textColor = color
		}
		if css, ok := tenantCtx.BrandingConfig["custom_css"]; ok {
			customCSS = css
		}
		if html, ok := tenantCtx.BrandingConfig["header_html"]; ok {
			headerHTML = html
		}
		if html, ok := tenantCtx.BrandingConfig["footer_html"]; ok {
			footerHTML = html
		}
	}

	branding := map[string]interface{}{
		"site_name":        siteName,
		"logo_url":         logoURL,
		"favicon_url":      faviconURL,
		"primary_color":    primaryColor,
		"secondary_color":  secondaryColor,
		"accent_color":     accentColor,
		"background_color": backgroundColor,
		"text_color":       textColor,
		"custom_css":       customCSS,
		"header_html":      headerHTML,
		"footer_html":      footerHTML,
	}

	c.JSON(http.StatusOK, types.SuccessResponse(branding, ""))
}

// UpdateBranding updates the branding configuration (admin only)
func (h *BrandingHandler) UpdateBranding(c *gin.Context) {
	// In production, this would validate admin permissions and update the database
	var req struct {
		SiteName        *string `json:"site_name"`
		LogoURL         *string `json:"logo_url"`
		FaviconURL      *string `json:"favicon_url"`
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
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	// In production, save to database and/or config
	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
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
	}, "Branding updated successfully"))
}
