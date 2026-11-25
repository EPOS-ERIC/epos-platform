package domain

import (
	"fmt"
	"strings"

	"github.com/EPOS-ERIC/epos-platform/packages/go/apperror"
	"github.com/google/uuid"
)

type DistributionType string

const (
	DistributionTypeWebService       DistributionType = "WEB_SERVICE"
	DistributionTypeDownloadableFile DistributionType = "DOWNLOADABLE_FILE"
)

func (t DistributionType) Valid() bool {
	return t == DistributionTypeDownloadableFile || t == DistributionTypeWebService
}

// DistributionInput contains caller-controlled distribution fields.
type DistributionInput struct {
	ResourceID       *uuid.UUID
	Title            string
	Description      string
	License          *string
	Type             DistributionType
	ByteSize         *int64
	WebService       WebServiceInput
	DownloadableFile *DownloadableFile
}

// Distribution is persisted distribution metadata.
type Distribution struct {
	ID               uuid.UUID
	ResourceID       uuid.UUID
	Title            string
	Description      string
	License          *string
	Type             DistributionType
	ByteSize         *int64
	WebService       WebService
	DownloadableFile *DownloadableFile
}

type DownloadableFile struct {
	URL    string
	Format string
}

// DistributionDownload is an authorized downloadable-file request target.
type DistributionDownload struct {
	ResourceID uuid.UUID
	VersionID  uuid.UUID
	URL        string
}

func (value *DistributionInput) Normalize() error {
	if isTypedNilWebService(value.WebService) {
		return apperror.Default(apperror.CodeInternal)
	}
	value.Title = strings.TrimSpace(value.Title)
	if value.Title == "" {
		return apperror.Field("title", "Title must not be blank.", nil)
	}
	value.Description = strings.TrimSpace(value.Description)
	if value.Description == "" {
		return apperror.Field("description", "Description must not be blank.", nil)
	}
	value.License = optionalTrimmedString(value.License)
	if value.WebService != nil && value.DownloadableFile != nil {
		return apperror.Field("type", "Choose either a downloadable file or a web service.", nil)
	}
	var expected DistributionType
	switch {
	case value.WebService != nil:
		expected = DistributionTypeWebService
	case value.DownloadableFile != nil:
		expected = DistributionTypeDownloadableFile
	default:
		return apperror.Field("type", "A delivery method is required.", nil)
	}
	if value.Type != "" && value.Type != expected {
		return apperror.Field("type", "Distribution type must match the selected delivery method.", nil)
	}
	value.Type = expected
	if value.DownloadableFile != nil {
		if err := normalizeHTTPURL(&value.DownloadableFile.URL, "downloadable_file.url"); err != nil {
			return err
		}
		if strings.TrimSpace(value.DownloadableFile.Format) == "" {
			return apperror.Field("downloadable_file.format", "Format must not be blank.", nil)
		}
	}
	if value.WebService != nil {
		if err := value.WebService.Normalize(); err != nil {
			return fmt.Errorf("validate web service: %w", apperror.PrefixTarget(err, "web_service"))
		}
	}
	return nil
}

func (value *Distribution) Normalize() (err error) {
	defer func() {
		if err != nil {
			err = internalValidationError(err)
		}
	}()
	value.License = optionalTrimmedString(value.License)
	if isTypedNilWebService(value.WebService) {
		return apperror.Default(apperror.CodeInternal)
	}
	if value.ID == uuid.Nil || strings.TrimSpace(value.Title) == "" {
		return fmt.Errorf("ID and title are required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if strings.TrimSpace(value.Description) == "" {
		return fmt.Errorf("distribution description is required: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if !value.Type.Valid() {
		return fmt.Errorf("distribution type is invalid: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	expected, err := value.DeliveryType()
	if err != nil {
		return fmt.Errorf("%w: %w", err, apperror.Default(apperror.CodeValidationFailed))
	}
	if value.Type != expected {
		return fmt.Errorf("distribution type does not match delivery method: %w", apperror.Default(apperror.CodeValidationFailed))
	}
	if value.DownloadableFile != nil {
		if err := normalizeHTTPURL(&value.DownloadableFile.URL, "downloadable_file.url"); err != nil {
			return err
		}
		// TODO this should become a mime type validation when the metadata is aligned
		if strings.TrimSpace(value.DownloadableFile.Format) == "" {
			return fmt.Errorf("downloadable file format is required: %w", apperror.Default(apperror.CodeValidationFailed))
		}
	}
	if value.WebService != nil {
		if err := value.WebService.Normalize(); err != nil {
			return fmt.Errorf("web service: %w", err)
		}
	}
	return nil
}

func (d Distribution) DeliveryType() (DistributionType, error) {
	if isTypedNilWebService(d.WebService) {
		return "", apperror.Default(apperror.CodeInternal)
	}
	switch {
	case d.WebService != nil && d.DownloadableFile == nil:
		return DistributionTypeWebService, nil
	case d.WebService == nil && d.DownloadableFile != nil:
		return DistributionTypeDownloadableFile, nil
	default:
		return "", fmt.Errorf("exactly one delivery method is required")
	}
}
