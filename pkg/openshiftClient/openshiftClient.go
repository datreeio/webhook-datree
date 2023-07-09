package openshiftClient

import (
	"context"
	"fmt"
	userClientV1 "github.com/openshift/client-go/user/clientset/versioned/typed/user/v1"
	"github.com/patrickmn/go-cache"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"time"
)

type OpenshiftClient struct {
	userClientV1 userClientV1.UserV1Interface
	cache        *cache.Cache
}

func NewOpenshiftClient() (*OpenshiftClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	userClientV1Instance, err := userClientV1.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &OpenshiftClient{
		userClientV1: userClientV1Instance,
		cache:        cache.New(1*time.Minute, 10*time.Minute),
	}, nil
}

type GroupsByUsers = map[string][]string

const groupsByUsersCacheKey = "groupsByUsers"

func (oc *OpenshiftClient) getGroupsByUsers() (GroupsByUsers, error) {
	// try to get from cache
	valueFromCache, found := oc.cache.Get(groupsByUsersCacheKey)
	if found {
		return valueFromCache.(GroupsByUsers), nil
	}

	// get from API
	groups, err := oc.userClientV1.Groups().List(context.TODO(), metav1.ListOptions{})
	fmt.Println("@@@@@@@@@@@@@@@@@@")
	fmt.Println(groups)
	fmt.Println(err)
	fmt.Println("@@@@@@@@@@@@@@@@@@")
	if err != nil {
		return nil, err
	}

	// convert from array of groups to map of users to groups
	groupsByUsers := make(GroupsByUsers)
	for _, group := range groups.Items {
		for _, user := range group.Users {
			groupsByUsers[user] = append(groupsByUsers[user], group.Name)
		}
	}

	// set to cache
	oc.cache.Set(groupsByUsersCacheKey, groupsByUsers, 1*time.Minute)
	return groupsByUsers, nil
}
func (oc *OpenshiftClient) GetGroupsUserBelongsTo(username string) ([]string, error) {
	groupsByUsers, err := oc.getGroupsByUsers()
	if err != nil {
		return nil, err
	}

	groups, found := groupsByUsers[username]
	if !found {
		return []string{}, nil
	}
	return groups, nil
}
