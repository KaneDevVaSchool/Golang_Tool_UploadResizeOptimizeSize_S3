package models

import "time"

// School đại diện cho 1 cơ sở vật lý của Hệ thống Trường Việt Mỹ.
type School struct {
	ID           int64     `db:"id" json:"id"`
	Name         string    `db:"name" json:"name"`
	Region       string    `db:"region" json:"region"`
	DisplayOrder int       `db:"display_order" json:"display_order"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

// Region: 3 khu vực trưng bày tranh (khác với 5 cơ sở vật lý thật).
const (
	RegionSaigon  = "saigon"
	RegionCanTho  = "cantho"
	RegionVungTau = "vungtau"
)

// RegionLabel trả về tên tiếng Việt hiển thị cho 1 region - dùng ở cả
// response API lẫn nơi cần label nhanh phía Go (vd log, seed script).
func RegionLabel(region string) string {
	switch region {
	case RegionSaigon:
		return "Sài Gòn"
	case RegionCanTho:
		return "Cần Thơ"
	case RegionVungTau:
		return "Vũng Tàu"
	default:
		return region
	}
}
