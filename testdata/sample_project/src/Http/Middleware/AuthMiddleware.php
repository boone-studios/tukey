<?php

namespace App\Http\Middleware;

use App\Domain\User;
use App\Domain\UserRepository;
use App\Domain\Enums\UserStatus;

class AuthMiddleware
{
    const TOKEN_PREFIX = 'Bearer ';
    const HEADER_KEY = 'Authorization';

    private UserRepository $userRepository;
    private array $excludedPaths = ['/health', '/login', '/register'];

    public function __construct(UserRepository $userRepository)
    {
        $this->userRepository = $userRepository;
    }

    public function handle(array $request, callable $next): array
    {
        $path = $request['path'] ?? '';
        if (in_array($path, $this->excludedPaths)) {
            return $next($request);
        }

        $token = $this->extractToken($request);
        if ($token === null) {
            return ['error' => 'Unauthorized', 'code' => 401];
        }

        $user = $this->authenticate($token);
        if ($user === null) {
            return ['error' => 'Invalid token', 'code' => 401];
        }

        if (!$user->getStatus()->canLogin()) {
            return ['error' => 'Account suspended', 'code' => 403];
        }

        $request['user'] = $user;
        return $next($request);
    }

    private function extractToken(array $request): ?string
    {
        $header = $request['headers'][self::HEADER_KEY] ?? '';
        if (!str_starts_with($header, self::TOKEN_PREFIX)) {
            return null;
        }
        return substr($header, strlen(self::TOKEN_PREFIX));
    }

    private function authenticate(string $token): ?User
    {
        $parts = explode(':', base64_decode($token));
        if (count($parts) !== 2) {
            return null;
        }
        return $this->userRepository->findByEmail($parts[0]);
    }

    public function requireAdmin(User $user): bool
    {
        return $user->isAdmin() && UserStatus::Active === $user->getStatus();
    }
}
