package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

// the most scuffed data storage solution of all time

var (
	ErrAlreadySleeping = fmt.Errorf("Channel already waiting for poll response")
	ErrNotSleeping     = fmt.Errorf("Channel is not currently sleeping")
	ErrNotPolling      = fmt.Errorf("Channel is not currently waiting for a poll response")
)

type SleepSession struct {
	Start            time.Time         `json:"end"`
	End              time.Time         `json:"start"`
	Duration         time.Duration     `json:"duration"`
	Quality          int               `json:"quality"`
	AdditionalFields map[string]string `json:"additional-fields"`

	ChannelID     string `json:"channel-id"`
	PollMessageID string `json:"poll-message-id"`
	Pending       bool   `json:"pending"`
}

type JSONDB struct {
	SleepSessions []SleepSession `json:"sleep-sessions"`
}

type DB struct {
	filename string
	Data     JSONDB
}

type DBInterface interface {
	StartSleepSession(channelID string, startTime time.Time, additionalFields map[string]string) error
	AddPollMessage(channelID string, pollMessageID string) error
	EndSleepSession(channelID string, endTime time.Time, qualityPoll int, additionalFields map[string]string) (SleepSession, error)
	DeletePendingSleepSessionPoll(channelID string) error

	GetChannelPending(channelID string) (startTime time.Time, pending bool)
	GetPollActive(pollMessageID string) bool

	GetAllSleepSessions() []SleepSession

	Save() error
	Load() error
}

func NewJSONDB(filename string) (DBInterface, error) {
	db := DB{
		filename: filename,
	}
	if err := db.Load(); err != nil {
		return nil, err
	}
	return &db, nil
}

func (db *DB) StartSleepSession(channelID string, startTime time.Time, additionalFields map[string]string) error {
	if _, pending := db.GetChannelPending(channelID); pending {
		return ErrAlreadySleeping
	}
	db.Data.SleepSessions = append(db.Data.SleepSessions, SleepSession{
		Start:            startTime,
		AdditionalFields: additionalFields,
		ChannelID:        channelID,
		Pending:          true,
	})
	return nil
}

func (db *DB) AddPollMessage(channelID string, pollMessageID string) error {
	for i, sleepSession := range db.Data.SleepSessions {
		if sleepSession.ChannelID == channelID && sleepSession.Pending {
			sessionData := sleepSession
			sessionData.PollMessageID = pollMessageID
			db.Data.SleepSessions[i] = sessionData
			return nil
		}
	}
	return ErrNotSleeping
}

func (db *DB) EndSleepSession(channelID string, endTime time.Time, qualityPoll int, additionalFields map[string]string) (SleepSession, error) {
	for i, sleepSession := range db.Data.SleepSessions {
		if sleepSession.ChannelID == channelID && sleepSession.Pending {
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
			return db.Data.SleepSessions[i], nil
		}
	}
	// logically we should only get here in the case where there's no pending sleep data for a given channel
	return SleepSession{}, ErrNotPolling
}

func (db *DB) DeletePendingSleepSessionPoll(channelID string) error {
	for i, sleepSession := range db.Data.SleepSessions {
		if sleepSession.ChannelID == channelID && sleepSession.Pending && sleepSession.PollMessageID != "" {
			// special case of only one item left to make sure indexing works
			if len(db.Data.SleepSessions) == 1 {
				db.Data.SleepSessions = make([]SleepSession, 0)
				return nil
			}
			db.Data.SleepSessions[i] = db.Data.SleepSessions[len(db.Data.SleepSessions)-1]
			db.Data.SleepSessions = db.Data.SleepSessions[:len(db.Data.SleepSessions)-1]
			return nil
		}
	}
	return ErrNotPolling
}

func (db *DB) GetChannelPending(channelID string) (startTime time.Time, pending bool) {
	for _, sleepSession := range db.Data.SleepSessions {
		if sleepSession.ChannelID == channelID && sleepSession.Pending {
			return sleepSession.Start, true
		}
	}
	return time.Time{}, false
}

func (db *DB) GetPollActive(pollMessageID string) bool {
	for _, sleepSession := range db.Data.SleepSessions {
		if sleepSession.PollMessageID == pollMessageID && sleepSession.Pending {
			return true
		} else if sleepSession.PollMessageID == pollMessageID {
			return false
		}
	}
	return false
}

func (db *DB) GetAllSleepSessions() []SleepSession {
	return db.Data.SleepSessions
}

func (db *DB) Save() error {
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

func (db *DB) Load() error {
	file, err := os.Open(db.filename)
	if err != nil {
		return err
	}
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
