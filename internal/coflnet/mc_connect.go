package coflnet

import (
	"encoding/json"
	"fmt"
	"github.com/Coflnet/coflnet-bot/internal/model"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/rs/zerolog/log"
)

type McConnectGetUserResponse struct {
	ExternalID string `json:"externalId"`
	Accounts   []struct {
		AccountUUID string `json:"accountUuid"`
		Verified    bool   `json:"verified"`
		UpdatedAt   string `json:"updatedAt"`
	} `json:"accounts"`
}

func UserMcConnect(userId int) (*model.User, error) {
	url := fmt.Sprintf("%s/Connect/user/%d", os.Getenv("MC_CONNECT_URL"), userId)

	response, err := http.DefaultClient.Get(url)
	if err != nil {
		log.Error().Err(err).Msgf("error getting user from mc connect, userId: %d", userId)
		return nil, err
	}

	defer func() {
		_ = response.Body.Close()
	}()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Error().Err(err).Msgf("error reading response body")
		return nil, err
	}

	var result McConnectGetUserResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Error().Err(err).Msgf("error unmarshalling response body")
		return nil, err
	}

	// if the user has no accounts, then the user probably doesn't exist
	// so return an error, to indicate that
	if len(result.Accounts) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	var uuids []string
	for _, u := range result.Accounts {

		// only add the verified ones
		if !u.Verified {
			continue
		}

		uuids = append(uuids, u.AccountUUID)
	}

	user := model.User{
		UserId:         userId,
		MinecraftUuids: uuids,
	}

	return &user, nil
}
