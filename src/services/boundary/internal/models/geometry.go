package models

type GeometryType string

const (
    GeometryPoint        GeometryType = "Point"
    GeometryPolygon      GeometryType = "Polygon"
    GeometryMultiPolygon GeometryType = "MultiPolygon"
)

func IsValidGeometryType(t string) bool {
    switch GeometryType(t) {
    case GeometryPoint, GeometryPolygon, GeometryMultiPolygon:
        return true
    default:
        return false
    }
} 