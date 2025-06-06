package domain

import (
	"time"
)

// Size represents the dimensions for an image variant
type Size struct {
	Width  int `yaml:"width" json:"width"`
	Height int `yaml:"height" json:"height"`
}

// ImageType represents a configured image type with its size variants
type ImageType struct {
	Name  string          `yaml:"name" json:"name"`
	Sizes map[string]Size `yaml:"sizes" json:"sizes"`
}

// ImageConfig represents the full image configuration from YAML
type ImageConfig struct {
	Images []ImageType `yaml:"images"`
}

// Image represents an image entity stored in the database
type Image struct {
	GUID           string    `json:"guid" db:"guid"`
	TypeID         int       `json:"typeId" db:"type_id"`
	TypeName       string    `json:"typeName" db:"type_name"`
	OwnerGUID      string    `json:"ownerGuid" db:"owner_guid"`
	SmallURL       string    `json:"smallUrl" db:"small_url"`
	MediumURL      string    `json:"mediumUrl" db:"medium_url"`
	LargeURL       string    `json:"largeUrl" db:"large_url"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time `json:"updatedAt" db:"updated_at"`
}

// UserImageResponse is the DTO for user image API responses
type UserImageResponse struct {
	UserGUID  string    `json:"userGuid"`
	ImageGUID string    `json:"imageGuid"`
	SmallURL  string    `json:"smallUrl"`
	MediumURL string    `json:"mediumUrl"`
	LargeURL  string    `json:"largeUrl"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// OrganizationImageResponse is the DTO for organization image API responses
type OrganizationImageResponse struct {
	OrganizationGUID string    `json:"organizationGuid"`
	ImageGUID        string    `json:"imageGuid"`
	SmallURL         string    `json:"smallUrl"`
	MediumURL        string    `json:"mediumUrl"`
	LargeURL         string    `json:"largeUrl"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// ImageRequest represents the metadata for an image upload request
type ImageRequest struct {
	ContentType string
	OwnerGUID   string
	TypeName    string
}

// ToUserImageResponse converts an Image entity to a UserImageResponse DTO
func (img *Image) ToUserImageResponse() UserImageResponse {
	return UserImageResponse{
		UserGUID:  img.OwnerGUID,
		ImageGUID: img.GUID,
		SmallURL:  img.SmallURL,
		MediumURL: img.MediumURL,
		LargeURL:  img.LargeURL,
		UpdatedAt: img.UpdatedAt,
	}
}

// ToOrganizationImageResponse converts an Image entity to an OrganizationImageResponse DTO
func (img *Image) ToOrganizationImageResponse() OrganizationImageResponse {
	return OrganizationImageResponse{
		OrganizationGUID: img.OwnerGUID,
		ImageGUID:        img.GUID,
		SmallURL:         img.SmallURL,
		MediumURL:        img.MediumURL,
		LargeURL:         img.LargeURL,
		UpdatedAt:        img.UpdatedAt,
	}
}
