package md2html

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

//go:embed css/*.css
var cssFS embed.FS

// CollectCSS returns the CSS content to embed in a <style> block.
// It always includes base.css, then for each role in usedRoles it includes
// {role}.css if available, otherwise includes default.css. The default.css
// is included at most once.
func CollectCSS(usedRoles []string) string {
	var sb strings.Builder

	// Always include base.css.
	if data, err := fs.ReadFile(cssFS, "css/base.css"); err == nil {
		sb.Write(data)
		sb.WriteByte('\n')
	}

	needDefault := false
	included := make(map[string]bool)

	for _, role := range usedRoles {
		role = strings.ToLower(role)
		if included[role] {
			continue
		}
		included[role] = true

		filename := fmt.Sprintf("css/%s.css", role)
		data, err := fs.ReadFile(cssFS, filename)
		if err != nil {
			// No dedicated CSS for this role -> will use default.
			needDefault = true
			continue
		}
		sb.Write(data)
		sb.WriteByte('\n')
	}

	if needDefault {
		if data, err := fs.ReadFile(cssFS, "css/default.css"); err == nil {
			sb.Write(data)
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

// HasRoleCSS reports whether a dedicated CSS file exists for the given role.
func HasRoleCSS(role string) bool {
	filename := fmt.Sprintf("css/%s.css", strings.ToLower(role))
	_, err := fs.ReadFile(cssFS, filename)
	return err == nil
}
