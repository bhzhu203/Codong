package interpreter

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/codong-lang/codong/stdlib/codongerror"
	"github.com/redis/go-redis/v9"
)

// RedisModuleObject is the singleton `redis` module.
type RedisModuleObject struct {
	client      *redis.Client
	connections map[string]*redis.Client
	defaultName string
	subs        []*redisSubscription
	mu          sync.Mutex
}

type redisSubscription struct {
	cancel  context.CancelFunc
	done    chan struct{}
	channel string
}

func (r *RedisModuleObject) Type() string    { return "module" }
func (r *RedisModuleObject) Inspect() string { return "<module:redis>" }

var redisModuleSingleton = &RedisModuleObject{
	connections: make(map[string]*redis.Client),
}

func (r *RedisModuleObject) getClient() *redis.Client {
	if r.defaultName != "" {
		if c, ok := r.connections[r.defaultName]; ok {
			return c
		}
	}
	return r.client
}

// RedisLockObject represents a distributed lock.
type RedisLockObject struct {
	key    string
	lockID string
	client *redis.Client
}

func (l *RedisLockObject) Type() string    { return "redis_lock" }
func (l *RedisLockObject) Inspect() string { return "<redis:lock>" }

// RedisSubscriptionObject represents an active subscription.
type RedisSubscriptionObject struct {
	sub *redisSubscription
}

func (s *RedisSubscriptionObject) Type() string    { return "redis_subscription" }
func (s *RedisSubscriptionObject) Inspect() string { return "<redis:subscription>" }

// evalRedisModuleMethod dispatches redis.xxx() calls.
func (interp *Interpreter) evalRedisModuleMethod(method string) Object {
	return &BuiltinFunction{
		Name: "redis." + method,
		Fn: func(i *Interpreter, args ...Object) Object {
			switch method {
			case "connect":
				return i.redisConnect(args)
			case "disconnect":
				return i.redisDisconnect()
			case "set":
				return i.redisSet(args)
			case "get":
				return i.redisGet(args)
			case "delete":
				return i.redisDelete(args)
			case "exists":
				return i.redisExists(args)
			case "expire":
				return i.redisExpire(args)
			case "ttl":
				return i.redisTTL(args)
			case "incr":
				return i.redisIncr(args)
			case "incr_by":
				return i.redisIncrBy(args)
			case "decr":
				return i.redisDecr(args)
			case "cache":
				return i.redisCache(args)
			case "invalidate":
				return i.redisInvalidate(args)
			case "invalidate_pattern":
				return i.redisInvalidatePattern(args)
			case "lock":
				return i.redisLock(args)
			case "publish":
				return i.redisPublish(args)
			case "subscribe":
				return i.redisSubscribe(args)
			case "using":
				return i.redisUsing(args)
			case "zadd":
				return i.redisZadd(args)
			case "zrange":
				return i.redisZrange(args)
			case "zrevrange":
				return i.redisZrevrange(args)
			case "zrank":
				return i.redisZrank(args)
			case "zrevrank":
				return i.redisZrevrank(args)
			case "zscore":
				return i.redisZscore(args)
			case "zincrby":
				return i.redisZincrby(args)
			case "rate_limiter":
				return i.redisRateLimiter(args)
			default:
				return newRuntimeError(codongerror.E10001_CONN_FAILED,
					fmt.Sprintf("unknown redis method: %s", method), "")
			}
		},
	}
}

func redisError(code, message, fix string) Object {
	return &ErrorObject{
		Error:     codongerror.New(code, message, codongerror.WithFix(fix)),
		IsRuntime: true,
	}
}

func getRedisClient() (*redis.Client, Object) {
	c := redisModuleSingleton.getClient()
	if c == nil {
		return nil, redisError(codongerror.E10001_CONN_FAILED,
			"no Redis connection", "call redis.connect() first")
	}
	return c, nil
}

func parseDuration(s string) time.Duration {
	// Handle "10m", "1h", "30s", "3600s" etc.
	d, err := time.ParseDuration(s)
	if err != nil {
		// Try with "s" suffix
		if !strings.ContainsAny(s, "smh") {
			d, _ = time.ParseDuration(s + "s")
		}
	}
	return d
}

func (i *Interpreter) redisConnect(args []Object) Object {
	if len(args) < 1 {
		return redisError(codongerror.E10001_CONN_FAILED,
			"redis.connect requires a URL", "redis.connect(\"redis://localhost:6379\")")
	}
	urlStr, ok := args[0].(*StringObject)
	if !ok {
		return redisError(codongerror.E10001_CONN_FAILED, "URL must be a string", "")
	}

	var connName string
	var opts *MapObject
	if len(args) > 1 {
		if m, ok := args[1].(*MapObject); ok {
			opts = m
			if v, ok := m.Entries["name"]; ok {
				connName = v.Inspect()
			}
		}
	}

	redisOpts, err := redis.ParseURL(urlStr.Value)
	if err != nil {
		return redisError(codongerror.E10001_CONN_FAILED,
			fmt.Sprintf("invalid Redis URL: %s", err.Error()),
			"check REDIS_URL format: redis://:password@host:6379/0")
	}

	// Apply pool config from options
	if opts != nil {
		if v, ok := opts.Entries["pool_size"]; ok {
			if n, ok := v.(*NumberObject); ok {
				redisOpts.PoolSize = int(n.Value)
			}
		}
		if v, ok := opts.Entries["min_idle"]; ok {
			if n, ok := v.(*NumberObject); ok {
				redisOpts.MinIdleConns = int(n.Value)
			}
		}
		if v, ok := opts.Entries["dial_timeout"]; ok {
			if s, ok := v.(*StringObject); ok {
				redisOpts.DialTimeout = parseDuration(s.Value)
			}
		}
		if v, ok := opts.Entries["read_timeout"]; ok {
			if s, ok := v.(*StringObject); ok {
				redisOpts.ReadTimeout = parseDuration(s.Value)
			}
		}
		if v, ok := opts.Entries["write_timeout"]; ok {
			if s, ok := v.(*StringObject); ok {
				redisOpts.WriteTimeout = parseDuration(s.Value)
			}
		}
	}

	client := redis.NewClient(redisOpts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return redisError(codongerror.E10001_CONN_FAILED,
			fmt.Sprintf("Redis connection failed: %s", err.Error()),
			"check REDIS_URL and network")
	}

	if connName != "" {
		if old, ok := redisModuleSingleton.connections[connName]; ok {
			old.Close()
		}
		redisModuleSingleton.connections[connName] = client
		if redisModuleSingleton.defaultName == "" {
			redisModuleSingleton.defaultName = connName
		}
	} else {
		if redisModuleSingleton.client != nil {
			redisModuleSingleton.client.Close()
		}
		redisModuleSingleton.client = client
	}

	return NULL_OBJ
}

func (i *Interpreter) redisDisconnect() Object {
	if redisModuleSingleton.client != nil {
		redisModuleSingleton.client.Close()
		redisModuleSingleton.client = nil
	}
	for _, c := range redisModuleSingleton.connections {
		c.Close()
	}
	redisModuleSingleton.connections = make(map[string]*redis.Client)
	// Cancel all subscriptions
	for _, sub := range redisModuleSingleton.subs {
		sub.cancel()
		<-sub.done
	}
	redisModuleSingleton.subs = nil
	return NULL_OBJ
}

func (i *Interpreter) redisUsing(args []Object) Object {
	if len(args) < 1 {
		return redisError(codongerror.E10001_CONN_FAILED, "redis.using requires a name", "")
	}
	name := args[0].(*StringObject).Value
	_, ok := redisModuleSingleton.connections[name]
	if !ok {
		return redisError(codongerror.E10001_CONN_FAILED,
			fmt.Sprintf("no Redis connection named '%s'", name), "")
	}
	// For now, return a map with the connection reference
	return &MapObject{
		Entries: map[string]Object{"_redis_name": &StringObject{Value: name}},
		Order:   []string{"_redis_name"},
	}
}

func (i *Interpreter) redisSet(args []Object) Object {
	client, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 2 {
		return redisError(codongerror.E10001_CONN_FAILED,
			"redis.set requires (key, value)", "")
	}
	key := args[0].Inspect()
	value := args[1].Inspect()

	ctx := context.Background()
	var ttl time.Duration

	// Check for named args (ttl, nx, xx)
	var nx, xx bool
	for _, a := range args[2:] {
		if m, ok := a.(*MapObject); ok {
			if v, ok := m.Entries["ttl"]; ok {
				ttl = parseDuration(v.Inspect())
			}
			if v, ok := m.Entries["nx"]; ok {
				if b, ok := v.(*BoolObject); ok {
					nx = b.Value
				}
			}
			if v, ok := m.Entries["xx"]; ok {
				if b, ok := v.(*BoolObject); ok {
					xx = b.Value
				}
			}
		}
	}

	if nx {
		ok, err := client.SetNX(ctx, key, value, ttl).Result()
		if err != nil {
			return redisError(codongerror.E10001_CONN_FAILED, err.Error(), "")
		}
		return nativeBoolToObject(ok)
	}
	if xx {
		ok, err := client.SetXX(ctx, key, value, ttl).Result()
		if err != nil {
			return redisError(codongerror.E10001_CONN_FAILED, err.Error(), "")
		}
		return nativeBoolToObject(ok)
	}

	err := client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return redisError(codongerror.E10001_CONN_FAILED, err.Error(), "")
	}
	return TRUE_OBJ
}

func (i *Interpreter) redisGet(args []Object) Object {
	client, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 1 {
		return redisError(codongerror.E10001_CONN_FAILED, "redis.get requires a key", "")
	}
	key := args[0].Inspect()

	ctx := context.Background()
	val, err := client.Get(ctx, key).Result()
	if err == redis.Nil {
		// Return default if provided
		if len(args) > 1 {
			return args[1]
		}
		return NULL_OBJ
	}
	if err != nil {
		return redisError(codongerror.E10001_CONN_FAILED, err.Error(), "")
	}
	return &StringObject{Value: val}
}

func (i *Interpreter) redisDelete(args []Object) Object {
	client, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 1 {
		return redisError(codongerror.E10001_CONN_FAILED, "redis.delete requires a key", "")
	}

	ctx := context.Background()

	// Handle batch delete with list
	if list, ok := args[0].(*ListObject); ok {
		keys := make([]string, len(list.Elements))
		for i, el := range list.Elements {
			keys[i] = el.Inspect()
		}
		n, err := client.Del(ctx, keys...).Result()
		if err != nil {
			return redisError(codongerror.E10001_CONN_FAILED, err.Error(), "")
		}
		return &NumberObject{Value: float64(n)}
	}

	key := args[0].Inspect()
	n, err := client.Del(ctx, key).Result()
	if err != nil {
		return redisError(codongerror.E10001_CONN_FAILED, err.Error(), "")
	}
	return &NumberObject{Value: float64(n)}
}

func (i *Interpreter) redisExists(args []Object) Object {
	client, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 1 {
		return FALSE_OBJ
	}
	ctx := context.Background()
	n, err := client.Exists(ctx, args[0].Inspect()).Result()
	if err != nil {
		return FALSE_OBJ
	}
	return nativeBoolToObject(n > 0)
}

func (i *Interpreter) redisExpire(args []Object) Object {
	client, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 2 {
		return FALSE_OBJ
	}
	ctx := context.Background()
	ttl := parseDuration(args[1].Inspect())
	ok, _ := client.Expire(ctx, args[0].Inspect(), ttl).Result()
	return nativeBoolToObject(ok)
}

func (i *Interpreter) redisTTL(args []Object) Object {
	client, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 1 {
		return &NumberObject{Value: -2}
	}
	ctx := context.Background()
	d, err := client.TTL(ctx, args[0].Inspect()).Result()
	if err != nil {
		return &NumberObject{Value: -2}
	}
	return &NumberObject{Value: d.Seconds()}
}

func (i *Interpreter) redisIncr(args []Object) Object {
	client, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 1 {
		return &NumberObject{Value: 0}
	}
	ctx := context.Background()
	n, err := client.Incr(ctx, args[0].Inspect()).Result()
	if err != nil {
		return redisError(codongerror.E10001_CONN_FAILED, err.Error(), "")
	}
	return &NumberObject{Value: float64(n)}
}

func (i *Interpreter) redisIncrBy(args []Object) Object {
	client, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 2 {
		return &NumberObject{Value: 0}
	}
	ctx := context.Background()
	amount := int64(args[1].(*NumberObject).Value)
	n, err := client.IncrBy(ctx, args[0].Inspect(), amount).Result()
	if err != nil {
		return redisError(codongerror.E10001_CONN_FAILED, err.Error(), "")
	}
	return &NumberObject{Value: float64(n)}
}

func (i *Interpreter) redisDecr(args []Object) Object {
	client, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 1 {
		return &NumberObject{Value: 0}
	}
	ctx := context.Background()
	n, err := client.Decr(ctx, args[0].Inspect()).Result()
	if err != nil {
		return redisError(codongerror.E10001_CONN_FAILED, err.Error(), "")
	}
	return &NumberObject{Value: float64(n)}
}

// singleflight-like mechanism for cache stampede protection
var (
	sfMu     sync.Mutex
	sfCalls  = make(map[string]*sfCall)
)

type sfCall struct {
	wg  sync.WaitGroup
	val Object
	err error
}

func singleflightDo(key string, fn func() (Object, error)) (Object, error) {
	sfMu.Lock()
	if call, ok := sfCalls[key]; ok {
		sfMu.Unlock()
		call.wg.Wait()
		return call.val, call.err
	}
	call := &sfCall{}
	call.wg.Add(1)
	sfCalls[key] = call
	sfMu.Unlock()

	call.val, call.err = fn()
	call.wg.Done()

	sfMu.Lock()
	delete(sfCalls, key)
	sfMu.Unlock()

	return call.val, call.err
}

const nullPlaceholder = "__codong_null__"

func (i *Interpreter) redisCache(args []Object) Object {
	client, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 2 {
		return redisError(codongerror.E10001_CONN_FAILED,
			"redis.cache requires (key, fn) or (key, fn, named_args)", "")
	}

	key := args[0].Inspect()

	// Find the function and TTL from args
	var fn *FunctionObject
	var ttl time.Duration = 10 * time.Minute
	cacheNull := true

	for _, a := range args[1:] {
		if f, ok := a.(*FunctionObject); ok {
			fn = f
		}
		if m, ok := a.(*MapObject); ok {
			if v, ok := m.Entries["ttl"]; ok {
				ttl = parseDuration(v.Inspect())
			}
			if v, ok := m.Entries["cache_null"]; ok {
				if b, ok := v.(*BoolObject); ok {
					cacheNull = b.Value
				}
			}
		}
	}

	if fn == nil {
		return redisError(codongerror.E10001_CONN_FAILED,
			"redis.cache requires a loader function", "")
	}

	ctx := context.Background()

	// Step 1: Check cache
	val, err := client.Get(ctx, key).Result()
	if err == nil {
		if val == nullPlaceholder {
			return NULL_OBJ
		}
		return &StringObject{Value: val}
	}

	// Step 2: Singleflight for stampede protection
	result, sfErr := singleflightDo(key, func() (Object, error) {
		// Double-check cache
		val, err := client.Get(ctx, key).Result()
		if err == nil {
			if val == nullPlaceholder {
				return NULL_OBJ, nil
			}
			return &StringObject{Value: val}, nil
		}

		// Call loader function
		data := i.applyFunction(fn, []Object{}, nil)

		// Handle errors from loader - don't cache errors
		if _, ok := data.(*ErrorObject); ok {
			return data, fmt.Errorf("loader error")
		}

		// Cache null protection
		if data == nil || data == NULL_OBJ {
			if cacheNull {
				nullTTL := ttl / 10
				if nullTTL > 30*time.Second {
					nullTTL = 30 * time.Second
				}
				client.Set(ctx, key, nullPlaceholder, nullTTL)
			}
			return NULL_OBJ, nil
		}

		// Cache the result
		client.Set(ctx, key, data.Inspect(), ttl)
		return data, nil
	})

	if sfErr != nil {
		// If it was a loader error, return the error object
		if result != nil {
			return result
		}
		return redisError(codongerror.E10001_CONN_FAILED, sfErr.Error(), "")
	}

	return result
}

func (i *Interpreter) redisInvalidate(args []Object) Object {
	client, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 1 {
		return NULL_OBJ
	}
	ctx := context.Background()
	client.Del(ctx, args[0].Inspect())
	return NULL_OBJ
}

func (i *Interpreter) redisInvalidatePattern(args []Object) Object {
	client, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 1 {
		return NULL_OBJ
	}
	ctx := context.Background()
	pattern := args[0].Inspect()

	// Use SCAN to find matching keys
	var cursor uint64
	for {
		keys, nextCursor, err := client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			break
		}
		if len(keys) > 0 {
			client.Del(ctx, keys...)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return NULL_OBJ
}

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Lua script for atomic lock release (verify UUID before delete)
var releaseLockScript = redis.NewScript(`
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	else
		return 0
	end
`)

func (i *Interpreter) redisLock(args []Object) Object {
	client, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 1 {
		return redisError(codongerror.E10004_LOCK_TIMEOUT, "redis.lock requires a key", "")
	}

	key := args[0].Inspect()
	lockTTL := 30 * time.Second
	timeout := 5 * time.Second
	retries := 3

	// Parse options
	for _, a := range args[1:] {
		if m, ok := a.(*MapObject); ok {
			if v, ok := m.Entries["ttl"]; ok {
				lockTTL = parseDuration(v.Inspect())
			}
			if v, ok := m.Entries["timeout"]; ok {
				timeout = parseDuration(v.Inspect())
			}
			if v, ok := m.Entries["retry"]; ok {
				if n, ok := v.(*NumberObject); ok {
					retries = int(n.Value)
				}
			}
		}
	}

	lockID := generateUUID()
	ctx := context.Background()
	deadline := time.Now().Add(timeout)

	for attempt := 0; attempt < retries && time.Now().Before(deadline); attempt++ {
		ok, err := client.SetNX(ctx, key, lockID, lockTTL).Result()
		if err != nil {
			return redisError(codongerror.E10004_LOCK_TIMEOUT, err.Error(), "")
		}
		if ok {
			lockObj := &RedisLockObject{key: key, lockID: lockID, client: client}
			return &MapObject{
				Entries: map[string]Object{
					"_lock":  lockObj,
					"key":    &StringObject{Value: key},
					"release": &BuiltinFunction{
						Name: "lock.release",
						Fn: func(_ *Interpreter, _ ...Object) Object {
							releaseLockScript.Run(ctx, client, []string{lockObj.key}, lockObj.lockID)
							return NULL_OBJ
						},
					},
				},
				Order: []string{"_lock", "key", "release"},
			}
		}
		time.Sleep(50 * time.Millisecond)
	}

	return NULL_OBJ // Lock acquisition failed
}

func (i *Interpreter) redisPublish(args []Object) Object {
	client, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 2 {
		return redisError(codongerror.E10001_CONN_FAILED,
			"redis.publish requires (channel, message)", "")
	}
	ctx := context.Background()
	channel := args[0].Inspect()
	message := args[1].Inspect()

	n, err := client.Publish(ctx, channel, message).Result()
	if err != nil {
		return redisError(codongerror.E10001_CONN_FAILED, err.Error(), "")
	}
	return &NumberObject{Value: float64(n)}
}

func (i *Interpreter) redisSubscribe(args []Object) Object {
	client, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 2 {
		return redisError(codongerror.E10001_CONN_FAILED,
			"redis.subscribe requires (channel, handler_fn)", "")
	}

	channel := args[0].Inspect()
	handler, ok := args[1].(*FunctionObject)
	if !ok {
		return redisError(codongerror.E10001_CONN_FAILED, "handler must be a function", "")
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	sub := &redisSubscription{cancel: cancel, done: done, channel: channel}

	go func() {
		defer close(done)
		backoff := 100 * time.Millisecond

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			pubsub := client.Subscribe(ctx, channel)

			for msg := range pubsub.Channel() {
				select {
				case <-ctx.Done():
					pubsub.Close()
					return
				default:
					i.applyFunction(handler, []Object{&StringObject{Value: msg.Payload}}, nil)
				}
			}

			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(backoff)
				if backoff < 30*time.Second {
					backoff *= 2
				}
			}
		}
	}()

	redisModuleSingleton.mu.Lock()
	redisModuleSingleton.subs = append(redisModuleSingleton.subs, sub)
	redisModuleSingleton.mu.Unlock()

	return &MapObject{
		Entries: map[string]Object{
			"channel": &StringObject{Value: channel},
			"unsubscribe": &BuiltinFunction{
				Name: "subscription.unsubscribe",
				Fn: func(_ *Interpreter, _ ...Object) Object {
					cancel()
					<-done
					return NULL_OBJ
				},
			},
		},
		Order: []string{"channel", "unsubscribe"},
	}
}

// Sorted set operations
func (i *Interpreter) redisZadd(args []Object) Object {
	client, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 2 {
		return &NumberObject{Value: 0}
	}
	ctx := context.Background()
	key := args[0].Inspect()

	members := []redis.Z{}
	if m, ok := args[1].(*MapObject); ok {
		for _, k := range m.Order {
			v := m.Entries[k]
			score := 0.0
			if n, ok := v.(*NumberObject); ok {
				score = n.Value
			}
			members = append(members, redis.Z{Score: score, Member: k})
		}
	}

	n, err := client.ZAdd(ctx, key, members...).Result()
	if err != nil {
		return redisError(codongerror.E10001_CONN_FAILED, err.Error(), "")
	}
	return &NumberObject{Value: float64(n)}
}

func (i *Interpreter) redisZrange(args []Object) Object {
	client, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 3 {
		return &ListObject{}
	}
	ctx := context.Background()
	key := args[0].Inspect()
	start := int64(args[1].(*NumberObject).Value)
	stop := int64(args[2].(*NumberObject).Value)

	vals, err := client.ZRangeWithScores(ctx, key, start, stop).Result()
	if err != nil {
		return &ListObject{}
	}

	elements := make([]Object, len(vals))
	for j, v := range vals {
		elements[j] = &MapObject{
			Entries: map[string]Object{
				"member": &StringObject{Value: v.Member.(string)},
				"score":  &NumberObject{Value: v.Score},
			},
			Order: []string{"member", "score"},
		}
	}
	return &ListObject{Elements: elements}
}

func (i *Interpreter) redisZrevrange(args []Object) Object {
	client, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 3 {
		return &ListObject{}
	}
	ctx := context.Background()
	key := args[0].Inspect()
	start := int64(args[1].(*NumberObject).Value)
	stop := int64(args[2].(*NumberObject).Value)

	vals, err := client.ZRevRangeWithScores(ctx, key, start, stop).Result()
	if err != nil {
		return &ListObject{}
	}

	elements := make([]Object, len(vals))
	for j, v := range vals {
		elements[j] = &MapObject{
			Entries: map[string]Object{
				"member": &StringObject{Value: v.Member.(string)},
				"score":  &NumberObject{Value: v.Score},
			},
			Order: []string{"member", "score"},
		}
	}
	return &ListObject{Elements: elements}
}

func (i *Interpreter) redisZrank(args []Object) Object {
	client, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 2 {
		return NULL_OBJ
	}
	ctx := context.Background()
	rank, err := client.ZRank(ctx, args[0].Inspect(), args[1].Inspect()).Result()
	if err != nil {
		return NULL_OBJ
	}
	return &NumberObject{Value: float64(rank)}
}

func (i *Interpreter) redisZrevrank(args []Object) Object {
	client, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 2 {
		return NULL_OBJ
	}
	ctx := context.Background()
	rank, err := client.ZRevRank(ctx, args[0].Inspect(), args[1].Inspect()).Result()
	if err != nil {
		return NULL_OBJ
	}
	return &NumberObject{Value: float64(rank)}
}

func (i *Interpreter) redisZscore(args []Object) Object {
	client, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 2 {
		return NULL_OBJ
	}
	ctx := context.Background()
	score, err := client.ZScore(ctx, args[0].Inspect(), args[1].Inspect()).Result()
	if err != nil {
		return NULL_OBJ
	}
	return &NumberObject{Value: score}
}

func (i *Interpreter) redisZincrby(args []Object) Object {
	client, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 3 {
		return NULL_OBJ
	}
	ctx := context.Background()
	incr := args[2].(*NumberObject).Value
	score, err := client.ZIncrBy(ctx, args[0].Inspect(), incr, args[1].Inspect()).Result()
	if err != nil {
		return redisError(codongerror.E10001_CONN_FAILED, err.Error(), "")
	}
	return &NumberObject{Value: score}
}

// Token bucket rate limiter using Lua script
var tokenBucketLua = redis.NewScript(`
	local key        = KEYS[1]
	local now        = tonumber(ARGV[1])
	local rate       = tonumber(ARGV[2])
	local burst      = tonumber(ARGV[3])
	local requested  = tonumber(ARGV[4])

	local data = redis.call("HMGET", key, "tokens", "last_refill")
	local tokens      = tonumber(data[1]) or burst
	local last_refill = tonumber(data[2]) or now

	local elapsed = (now - last_refill) / 1000.0
	tokens = math.min(burst, tokens + elapsed * rate)

	if tokens >= requested then
		tokens = tokens - requested
		redis.call("HMSET", key, "tokens", tokens, "last_refill", now)
		redis.call("PEXPIRE", key, math.ceil(burst / rate * 1000) + 1000)
		return {1, math.floor(tokens), 0}
	else
		local wait_ms = math.ceil((requested - tokens) / rate * 1000)
		redis.call("HMSET", key, "tokens", tokens, "last_refill", now)
		redis.call("PEXPIRE", key, math.ceil(burst / rate * 1000) + 1000)
		return {0, math.floor(tokens), wait_ms}
	end
`)

func (i *Interpreter) redisRateLimiter(args []Object) Object {
	_, errObj := getRedisClient()
	if errObj != nil {
		return errObj
	}
	if len(args) < 1 {
		return redisError(codongerror.E10001_CONN_FAILED,
			"redis.rate_limiter requires config", "")
	}

	opts, ok := args[0].(*MapObject)
	if !ok {
		return redisError(codongerror.E10001_CONN_FAILED, "config must be a map", "")
	}

	keyPrefix := "ratelimit"
	rate := 100.0
	burst := 200.0

	if v, ok := opts.Entries["key"]; ok {
		keyPrefix = v.Inspect()
	}
	if v, ok := opts.Entries["rate"]; ok {
		if n, ok := v.(*NumberObject); ok {
			rate = n.Value
		}
	}
	if v, ok := opts.Entries["burst"]; ok {
		if n, ok := v.(*NumberObject); ok {
			burst = n.Value
		}
	}

	capturedRate := rate
	capturedBurst := burst
	capturedPrefix := keyPrefix

	return &MapObject{
		Entries: map[string]Object{
			"allow": &BuiltinFunction{
				Name: "rate_limiter.allow",
				Fn: func(_ *Interpreter, fnArgs ...Object) Object {
					client := redisModuleSingleton.getClient()
					if client == nil {
						// Fail open
						return &MapObject{
							Entries: map[string]Object{
								"allowed":        TRUE_OBJ,
								"remaining":      &NumberObject{Value: -1},
								"retry_after_ms": &NumberObject{Value: 0},
							},
							Order: []string{"allowed", "remaining", "retry_after_ms"},
						}
					}

					userKey := ""
					if len(fnArgs) > 0 {
						userKey = fnArgs[0].Inspect()
					}
					key := fmt.Sprintf("%s:%s", capturedPrefix, userKey)
					now := time.Now().UnixMilli()
					ctx := context.Background()

					result, err := tokenBucketLua.Run(ctx, client,
						[]string{key}, now, capturedRate, capturedBurst, 1).Int64Slice()

					if err != nil {
						// Fail open on Redis error
						return &MapObject{
							Entries: map[string]Object{
								"allowed":        TRUE_OBJ,
								"remaining":      &NumberObject{Value: -1},
								"retry_after_ms": &NumberObject{Value: 0},
							},
							Order: []string{"allowed", "remaining", "retry_after_ms"},
						}
					}

					return &MapObject{
						Entries: map[string]Object{
							"allowed":        nativeBoolToObject(result[0] == 1),
							"remaining":      &NumberObject{Value: float64(result[1])},
							"retry_after_ms": &NumberObject{Value: float64(result[2])},
						},
						Order: []string{"allowed", "remaining", "retry_after_ms"},
					}
				},
			},
		},
		Order: []string{"allow"},
	}
}
