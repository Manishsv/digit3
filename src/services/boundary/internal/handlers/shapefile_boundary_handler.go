package handlers

import (
	"boundary/internal/config"
	"boundary/internal/models"
	"boundary/internal/service"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	shp "github.com/jonas-p/go-shp"
)

type ShapefileBoundaryCreateRequest struct {
	TenantID        string   `json:"tenantId"`
	FileStoreIds    []string `json:"fileStoreIds"`
	UniqueCodeField string   `json:"uniqueCodeField"`
}

type FileStoreResponse struct {
	Files map[string]string `json:"files"` // fileStoreId -> signedUrl
}

func ShapefileBoundaryCreateHandler(boundaryService service.BoundaryService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var req ShapefileBoundaryCreateRequest
		if err := ctx.ShouldBindJSON(&req); err != nil {
			errorResponse(ctx, http.StatusBadRequest, "BAD_REQUEST", err.Error(), "Invalid request payload", nil)
			return
		}

		tenantID := ctx.GetHeader("X-Tenant-ID")
		clientID := ctx.GetHeader("X-Client-Id")
		if tenantID == "" || clientID == "" {
			errorResponse(ctx, http.StatusBadRequest, "BAD_REQUEST", "Missing X-Tenant-ID or X-Client-Id header", "Missing X-Tenant-ID or X-Client-Id header", nil)
			return
		}

		if req.TenantID == "" || len(req.FileStoreIds) == 0 {
			errorResponse(ctx, http.StatusBadRequest, "BAD_REQUEST", "tenantId and fileStoreIds are required", "tenantId and fileStoreIds are required", nil)
			return
		}
		// Overwrite tenantId from header
		req.TenantID = tenantID

		cfg := config.LoadConfig()
		basePath := cfg.Filestore.BasePath
		endpoint := cfg.Filestore.Endpoint

		// 1. Download files using new filestore API
		tempDir, err := ioutil.TempDir("", "shapefile_upload_")
		if err != nil {
			errorResponse(ctx, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Failed to create temp dir: "+err.Error(), "Failed to create temp dir", nil)
			return
		}
		defer os.RemoveAll(tempDir) // Clean up temp dir at the end

		var mainShpPath string
		var baseName string
		for _, fileStoreId := range req.FileStoreIds {
			url := basePath + endpoint + "/" + fileStoreId + "?tenantId=" + req.TenantID
			resp, err := http.Get(url)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to download file: " + err.Error()})
				return
			}
			if resp.StatusCode != 200 {
				body, _ := ioutil.ReadAll(resp.Body)
				resp.Body.Close()
				ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Filestore API error: %s", string(body))})
				return
			}

			// Get filename from Content-Disposition header
			filename := ""
			cd := resp.Header.Get("Content-Disposition")
			if cd != "" {
				parts := strings.Split(cd, "filename=")
				if len(parts) > 1 {
					filename = strings.Trim(parts[1], `"`)
				}
			}
			if filename == "" {
				filename = fileStoreId // fallback
			}
			ext := filepath.Ext(filename)
			if ext == ".shp" && baseName == "" {
				baseName = strings.TrimSuffix(filename, ext)
			}
			localPath := filepath.Join(tempDir, filename)
			out, err := os.Create(localPath)
			if err != nil {
				resp.Body.Close()
				errorResponse(ctx, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Failed to create file: "+err.Error(), "Failed to create file", nil)
				return
			}
			_, err = io.Copy(out, resp.Body)
			resp.Body.Close()
			out.Close()
			if err != nil {
				errorResponse(ctx, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Failed to save file: "+err.Error(), "Failed to save file", nil)
				return
			}
		}

		if baseName == "" {
			errorResponse(ctx, http.StatusBadRequest, "BAD_REQUEST", "No .shp file found in uploaded files", "No .shp file found in uploaded files", nil)
			return
		}

		// Rename all shapefile parts to have the same base name
		parts := []string{".shp", ".shx", ".dbf", ".prj"}
		for _, ext := range parts {
			var found string
			files, _ := ioutil.ReadDir(tempDir)
			for _, file := range files {
				if strings.HasSuffix(strings.ToLower(file.Name()), ext) {
					found = file.Name()
					break
				}
			}
			if found != "" {
				newName := baseName + ext
				if found != newName {
					os.Rename(filepath.Join(tempDir, found), filepath.Join(tempDir, newName))
				}
			}
		}

		mainShpPath = filepath.Join(tempDir, baseName+".shp")

		// 2. Validate shapefile
		shape, err := shp.Open(mainShpPath)
		if err != nil {
			errorResponse(ctx, http.StatusBadRequest, "BAD_REQUEST", "Invalid shapefile: "+err.Error(), "Invalid shapefile", nil)
			return
		}
		defer shape.Close()
		fields := shape.Fields()
		nRecords := 0
		boundaries := []models.Boundary{}
		for shape.Next() {
			idx, geom := shape.Shape()
			props := make(map[string]interface{})
			for i, f := range fields {
				val := shape.ReadAttribute(idx, i)
				props[f.String()] = val
			}
			var code string
			if req.UniqueCodeField != "" {
				v, ok := props[req.UniqueCodeField]
				if !ok {
					v, ok = props[strings.ToUpper(req.UniqueCodeField)]
				}
				if !ok {
					v, ok = props[strings.ToLower(req.UniqueCodeField)]
				}
				if ok && v != nil && fmt.Sprintf("%v", v) != "" {
					code = strings.ToUpper(fmt.Sprintf("%v", v))
				}
			}
			if code == "" {
				fieldsToTry := []string{"name", "admin", "name_en"}
				for _, field := range fieldsToTry {
					v, ok := props[field]
					if !ok {
						v, ok = props[strings.ToUpper(field)]
					}
					if !ok {
						v, ok = props[strings.ToLower(field)]
					}
					if ok && v != nil && fmt.Sprintf("%v", v) != "" {
						code = strings.ToUpper(fmt.Sprintf("%v", v))
						break
					}
				}
			}
			// Remove null bytes from code
			code = strings.ReplaceAll(code, "\x00", "")
			if code == "" {
				code = fmt.Sprintf("BOUNDARY-%d", idx)
			}
			geojson, err := shapefileToGeoJSON(geom)
			if err != nil {
				continue // skip invalid geometry
			}
			geojsonBytes, _ := json.Marshal(geojson)
			// Geometry type validation
			geomType, ok := geojson["type"].(string)
			if !ok || !models.IsValidGeometryType(geomType) {
				errorResponse(ctx, http.StatusBadRequest, "BAD_REQUEST", "Invalid geometry type in shapefile", "Allowed types: Point, Polygon, MultiPolygon", nil)
				return
			}
			boundaries = append(boundaries, models.Boundary{
				TenantID:          req.TenantID,
				Code:              code,
				Geometry:          geojsonBytes,
				AdditionalDetails: json.RawMessage([]byte("{}")),
			})
			nRecords++
		}
		if nRecords == 0 {
			errorResponse(ctx, http.StatusBadRequest, "BAD_REQUEST", "No valid records found in shapefile", "No valid records found in shapefile", nil)
			return
		}

		// 3. Call boundary service method directly
		boundaryReq := &models.BoundaryRequest{
			Boundary: boundaries,
		}
		err = boundaryService.Create(ctx.Request.Context(), boundaryReq, tenantID, clientID)
		if err != nil {
			errorResponse(ctx, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "Failed to create boundaries: "+err.Error(), "Failed to create boundaries", nil)
			return
		}

		ctx.JSON(http.StatusOK, gin.H{"message": "Boundaries created successfully", "count": nRecords})
	}
}

// shapefileToGeoJSON converts a go-shp Shape to a GeoJSON geometry object
func shapefileToGeoJSON(shape shp.Shape) (map[string]interface{}, error) {
	switch s := shape.(type) {
	case *shp.Point:
		return map[string]interface{}{
			"type":        "Point",
			"coordinates": []float64{s.X, s.Y},
		}, nil
	case *shp.PolyLine:
		coords := [][]float64{}
		for _, pt := range s.Points {
			coords = append(coords, []float64{pt.X, pt.Y})
		}
		return map[string]interface{}{
			"type":        "LineString",
			"coordinates": coords,
		}, nil
	case *shp.Polygon:
		coords := [][]float64{}
		for _, pt := range s.Points {
			coords = append(coords, []float64{pt.X, pt.Y})
		}
		return map[string]interface{}{
			"type":        "Polygon",
			"coordinates": []interface{}{coords},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported shape type: %T", shape)
	}
}
