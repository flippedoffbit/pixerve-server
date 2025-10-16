package job

import (
	"fmt"
	"pixerve/models"
	"pixerve/utils"
)

type combinedJob struct {
	ConversionJobs  []models.ConversionJob
	WriterJobs      []models.WriterJob
	CallbackURL     string
	CallbackHeaders map[string]string
	Priority        int
	KeepOriginal    bool
	SubDir          string
}

func ParseTokenIntoJobs(tokenString string) (combinedJob, error) {
	claims, err := utils.VerifyPixerveJWT(tokenString, utils.VerifyConfig{})
	if err != nil {
		return combinedJob{}, fmt.Errorf("failed to verify JWT: %w", err)
	}
	return parseClaimsIntoJobs(claims)
}

func ParseTokenIntoJobsFromClaims(claims *models.PixerveJWT) (combinedJob, error) {
	return parseClaimsIntoJobs(claims)
}

func parseClaimsIntoJobs(task *models.PixerveJWT) (combinedJob, error) {
	if task == nil {
		return combinedJob{}, fmt.Errorf("task is nil")
	}

	var encodeJobs []models.ConversionJob = make([]models.ConversionJob, 0)
	var writerJobs []models.WriterJob = make([]models.WriterJob, 0)

	for format, spec := range task.Job.Formats {
		for _, size := range spec.Sizes {
			var length, width int
			if len(size) == 1 {
				length = size[0]
				width = size[0]
			} else if len(size) == 2 {
				width = size[0]
				length = size[1]
			} else {
				return combinedJob{}, fmt.Errorf("invalid size specification: %v", size)
			}
			encodeJobs = append(encodeJobs, models.ConversionJob{
				Encoder: format,
				Length:  length,
				Width:   width,
				Quality: spec.Settings.Quality,
				Speed:   spec.Settings.Speed,
			})
		}
	}

	for storageType, key := range task.Job.StorageKeys {
		writerJobs = append(writerJobs, models.WriterJob{
			Type:        storageType,
			Credentials: map[string]string{"key": key},
		})
	}

	if task.Job.DirectHost {
		writerJobs = append(writerJobs, models.WriterJob{
			Type:        "directServe",
			Credentials: map[string]string{},
		})
	}

	if task.Job.KeepOriginal {
		encodeJobs = append(encodeJobs, models.ConversionJob{
			Encoder: "copy",
			Length:  0,   // Not applicable for copy
			Width:   0,   // Not applicable for copy
			Quality: 100, // Not applicable for copy
			Speed:   0,   // Not applicable for copy
		})
	}

	return combinedJob{
		ConversionJobs:  encodeJobs,
		WriterJobs:      writerJobs,
		CallbackURL:     task.Job.CompletionCallback,
		CallbackHeaders: task.Job.CallbackHeaders,
		Priority:        task.Job.Priority,
		KeepOriginal:    task.Job.KeepOriginal,
		SubDir:          task.Job.SubDir,
	}, nil
}
