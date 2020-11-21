package doremy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

// the most scuffed data storage solution of all time

var (
	errAlreadySleeping = fmt.Errorf("Channel already waiting for poll response")
	errNotSleeping     = fmt.Errorf("Channel is not currently sleeping")
	errNotPolling      = fmt.Errorf("Channel is not currently waiting for a poll response")
	errPollNotFound    = fmt.Errorf("Poll message not found")
)

type sleepSessionStruct struct {
	Start            time.Time         `json:"end"`
	End              time.Time         `json:"start"`
	Duration         time.Duration     `json:"duration"`
	Quality          int               `json:"quality"`
	AdditionalFields map[string]string `json:"additional-fields"`

	ChannelID     string `json:"channel-id"`
	PollMessageID string `json:"poll-message-id"`
	Pending       bool   `json:"pending"`
}

type jsonDB struct {
	SleepSessions []sleepSessionStruct `json:"sleep-sessions"`
}

type dbStruct struct {
	filename string
	Data     jsonDB
}

type dbInterface interface {
	StartSleepSession(channelID string, startTime time.Time, additionalFields map[string]string) error
	AddPollMessage(channelID string, pollMessageID string) error
	EndSleepSession(channelID string, endTime time.Time, qualityPoll int, additionalFields map[string]string) (sleepSessionStruct, error)
	UpdatePollingData(pollMessageID string, quality int) (sleepSessionStruct, error)
	DeletePendingSleepSession(channelID string) error

	GetChannelPending(channelID string) (sleepSession sleepSessionStruct, pending bool)
	GetPollActive(pollMessageID string) bool
	GetAllUnaddedPolls() []sleepSessionStruct

	GetAllSleepSessions() []sleepSessionStruct

	Save() error
	Load() error
}

func newJSONDB(filename string) (dbInterface, error) {
	db := dbStruct{
		filename: filename,
	}
	if err := db.Load(); err != nil {
		return nil, err
	}
	return &db, nil
}

func (db *dbStruct) StartSleepSession(channelID string, startTime time.Time, additionalFields map[string]string) error {
	if _, pending := db.GetChannelPending(channelID); pending {
		return errAlreadySleeping
	}
	db.Data.SleepSessions = append(db.Data.SleepSessions, sleepSessionStruct{
		Start:            startTime,
		AdditionalFields: additionalFields,
		ChannelID:        channelID,
		Pending:          true,
	})
	return nil
}

func (db *dbStruct) AddPollMessage(channelID string, pollMessageID string) error {
	for i, sleepSession := range db.Data.SleepSessions {
		if sleepSession.ChannelID == channelID && sleepSession.Pending {
			sessionData := sleepSession
			sessionData.PollMessageID = pollMessageID
			db.Data.SleepSessions[i] = sessionData
			return nil
		}
	}
	return errNotSleeping
}

func (db *dbStruct) EndSleepSession(channelID string, endTime time.Time, qualityPoll int, additionalFields map[string]string) (sleepSessionStruct, error) {
	for i, sleepSession := range db.Data.SleepSessions {
		if sleepSession.ChannelID == channelID && sleepSession.Pending && sleepSession.PollMessageID != "" {
			updatedAdditionalFields := sleepSession.AdditionalFields
			for k, v := range additionalFields {
				updatedAdditionalFields[k] = v
			}
			sessionData := sleepSession
			sessionData.End = endTime
			sessionData.Duration = endTime.Sub(sleepSession.Start)
			sessionData.Quality = qualityPoll
			sessionData.AdditionalFields = updatedAdditionalFields
			sessionData.Pending = false
			db.Data.SleepSessions[i] = sessionData
			return db.Data.SleepSessions[i], nil
		}
	}
	// logically we should only get here in the case where there's no pending sleep data for a given channel
	return sleepSessionStruct{}, errNotPolling
}

func (db *dbStruct) UpdatePollingData(pollMessageID string, quality int) (sleepSessionStruct, error) {
	for i, sleepSession := range db.Data.SleepSessions {
		if sleepSession.PollMessageID == pollMessageID && pollMessageID != "" {
			updatedSleepSession := sleepSession
			updatedSleepSession.Quality = quality
			db.Data.SleepSessions[i] = updatedSleepSession
			return db.Data.SleepSessions[i], nil
		}
	}
	return sleepSessionStruct{}, errPollNotFound
}

func (db *dbStruct) DeletePendingSleepSession(channelID string) error {
	for i, sleepSession := range db.Data.SleepSessions {
		if sleepSession.ChannelID == channelID && sleepSession.Pending {
			// special case of only one item left to make sure indexing works
			if len(db.Data.SleepSessions) == 1 {
				db.Data.SleepSessions = make([]sleepSessionStruct, 0)
				return nil
			}
			db.Data.SleepSessions[i] = db.Data.SleepSessions[len(db.Data.SleepSessions)-1]
			db.Data.SleepSessions = db.Data.SleepSessions[:len(db.Data.SleepSessions)-1]
			return nil
		}
	}
	return errNotSleeping
}

func (db *dbStruct) GetChannelPending(channelID string) (sleepSession sleepSessionStruct, pending bool) {
	for _, sleepSession := range db.Data.SleepSessions {
		if sleepSession.ChannelID == channelID && sleepSession.Pending {
			return sleepSession, true
		}
	}
	return sleepSessionStruct{}, false
}

func (db *dbStruct) GetPollActive(pollMessageID string) bool {
	for _, sleepSession := range db.Data.SleepSessions {
		if sleepSession.PollMessageID == pollMessageID && sleepSession.Pending {
			return true
		} else if sleepSession.PollMessageID == pollMessageID {
			return false
		}
	}
	return false
}

func (db *dbStruct) GetAllUnaddedPolls() []sleepSessionStruct {
	sessions := make([]sleepSessionStruct, 0)
	for _, sleepSession := range db.Data.SleepSessions {
		if sleepSession.PollMessageID == "" && sleepSession.Pending {
			sessions = append(sessions, sleepSession)
		}
	}
	return sessions
}

func (db *dbStruct) GetAllSleepSessions() []sleepSessionStruct {
	return db.Data.SleepSessions
}

func (db *dbStruct) Save() error {
	marshalled, err := json.Marshal(db.Data)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(db.filename, marshalled, 0644)
	if err != nil {
		return err
	}
	return nil
}

func (db *dbStruct) Load() error {
	file, err := os.Open(db.filename)
	if err != nil {
		return err
	}
	defer file.Close()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &db.Data)
	if err != nil {
		return err
	}
	return nil
}
