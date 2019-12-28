package redis-util

import (
	"strings"

	"github.com/go-redis/redis"
)

var delIfValueScript = strings.Join([]string{
	"local value = redis.call('get',KEYS[1])	",
	"if value == ARGV[1] then					",
	"	redis.call('del', KEYS[1])				",
	"end										",
	"return value								",
}, "\n")

// DelIfValue will delete a key if its value matches the given value. It returns the same as Get for the key
func DelIfValue(client *redis.Client, key string, value string) *redis.Cmd {
	return client.Eval(delIfValueScript, []string{key}, []string{value})
}
