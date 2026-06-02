<?php

namespace App\Infrastructure\Caching;

use App\Domain\Contracts\Cacheable;

class RedisCache implements Cacheable
{
    const DEFAULT_TTL = 3600;

    private \Redis $redis;
    private string $prefix;
    private static array $localCache = [];

    public function __construct(string $host, int $port, string $prefix)
    {
        $this->redis = new \Redis();
        $this->redis->connect($host, $port);
        $this->prefix = $prefix;
    }

    public function get(string $key): mixed
    {
        $prefixed = $this->prefix . $key;
        if (isset(self::$localCache[$prefixed])) {
            return self::$localCache[$prefixed];
        }
        $value = $this->redis->get($prefixed);
        return $value !== false ? unserialize($value) : null;
    }

    public function set(string $key, mixed $value, int $ttl = self::DEFAULT_TTL): void
    {
        $prefixed = $this->prefix . $key;
        self::$localCache[$prefixed] = $value;
        $this->redis->setex($prefixed, $ttl, serialize($value));
    }

    public function getCacheKey(): string
    {
        return $this->prefix . 'cache';
    }

    public function getCacheTtl(): int
    {
        return self::DEFAULT_TTL;
    }

    public function clearCache(string $key): void
    {
        $prefixed = $this->prefix . $key;
        unset(self::$localCache[$prefixed]);
        $this->redis->del($prefixed);
    }

    public function warmup(Cacheable $entity): void
    {
        $this->set($entity->getCacheKey(), $entity, $entity->getCacheTtl());
    }

    public function flush(): void
    {
        self::$localCache = [];
        $this->redis->flushDB();
    }
}
