package domain

import (
	"time"

	"github.com/google/uuid"
)

// Size represents the dimensions for an image variant
type Size struct {
	Width  int `json:"width" yaml:"width"`
	Height int `json:"height" yaml:"height"` // 0 means auto-scale height proportionally
}

// SizeSet is a map of named sizes (small, medium, large) to their dimensions
type SizeSet map[string]Size

// ImageType represents a category of images with specific size configurations
type ImageType struct {
	Name  string  `json:"name" yaml:"name"`
	Sizes SizeSet `json:"sizes" yaml:"sizes"`
}

// ImageConfig holds the configuration for all image types
type ImageConfig struct {
	Types []ImageType `json:"images" yaml:"images"`
}

// Image represents a stored image with its metadata and URLs
type Image struct {
	GUID           uuid.UUID `json:"guid" db:"guid"`
	OwnerGUID      uuid.UUID `json:"ownerGuid" db:"owner_guid"` // User or Organization GUID
	TypeName       string    `json:"typeName" db:"type_name"`   // "user", "organization", etc.
	SmallURL       string    `json:"smallUrl" db:"small_url"`
	MediumURL      string    `json:"mediumUrl" db:"medium_url"`
	LargeURL       string    `json:"largeUrl" db:"large_url"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time `json:"updatedAt" db:"updated_at"`
	ContentType    string    `json:"contentType,omitempty" db:"content_type"`
	OriginalWidth  int       `json:"originalWidth,omitempty" db:"original_width"`
	OriginalHeight int       `json:"originalHeight,omitempty" db:"original_height"`
}

// UserImage is a specialized view of Image for user images
type UserImage struct {
	UserGUID  uuid.UUID `json:"userGuid"`
	ImageGUID uuid.UUID `json:"imageGuid"`
	SmallURL  string    `json:"smallUrl"`
	MediumURL string    `json:"mediumUrl"`
	LargeURL  string    `json:"largeUrl"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// OrganizationImage is a specialized view of Image for organization images
type OrganizationImage struct {
	OrganizationGUID uuid.UUID `json:"organizationGuid"`
	ImageGUID        uuid.UUID `json:"imageGuid"`
	SmallURL         string    `json:"smallUrl"`
	MediumURL        string    `json:"mediumUrl"`
	LargeURL         string    `json:"largeUrl"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// NewImage creates a new Image instance with generated GUID and timestamps
func NewImage(ownerGUID uuid.UUID, typeName string) *Image {
	now := time.Now().UTC()
	return &Image{
		GUID:      uuid.New(),
		OwnerGUID: ownerGUID,
		TypeName:  typeName,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// ToUserImage converts an Image to a UserImage view
func (i *Image) ToUserImage() *UserImage {
	return &UserImage{
		UserGUID:  i.OwnerGUID,
		ImageGUID: i.GUID,
		SmallURL:  i.SmallURL,
		MediumURL: i.MediumURL,
		LargeURL:  i.LargeURL,
		UpdatedAt: i.UpdatedAt,
	}
}

// ToOrganizationImage converts an Image to an OrganizationImage view
func (i *Image) ToOrganizationImage() *OrganizationImage {
	return &OrganizationImage{
		OrganizationGUID: i.OwnerGUID,
		ImageGUID:        i.GUID,
		SmallURL:         i.SmallURL,
		MediumURL:        i.MediumURL,
		LargeURL:         i.LargeURL,
		UpdatedAt:        i.UpdatedAt,
	}
}

// GetImageTypeByName returns the ImageType with the given name from the config
func GetImageTypeByName(config *ImageConfig, name string) (*ImageType, bool) {
	for _, t := range config.Types {
		if t.Name == name {
			return &t, true
		}
	}
	return nil, false
}
