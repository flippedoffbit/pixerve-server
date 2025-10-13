package job

import (
	"pixerve/models"
	"pixerve/utils"
)

type combinedJob struct {
	ConversionJobs []models.ConversionJob
	WriterJobs     []models.WriterJob
}

func ParseToken(tokenString string) (combinedJob, error) {
	task, err := utils.VerifyPixerveJWT(tokenString, utils.VerifyConfig{})

	if err != nil {
		return combinedJob{}, err
	}

	var encodeJobs []models.ConversionJob = make([]models.ConversionJob, 0)
	var writerJobs []models.WriterJob = make([]models.WriterJob, 0)

	// for destination, keys := range task.Job.StorageKeys {

	// }
	if task.Job.KeepOriginal {

	}
	// Placeholder implementation

	return combinedJob{
		ConversionJobs: encodeJobs,
		WriterJobs:     writerJobs,
	}, nil
}
