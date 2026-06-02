<?php

namespace App\Http\Controllers;

use App\Application\UserService;
use App\Application\Commands\CreateUserCommand;
use App\Application\Handlers\CreateUserHandler;
use App\Domain\Enums\UserStatus;

class UserController
{
    private UserService $userService;

    public function __construct(UserService $userService)
    {
        $this->userService = $userService;
    }

    public function show(int $id): array
    {
        $user = $this->userService->getUserById($id);
        if ($user === null) {
            return ['error' => 'User not found', 'code' => 404];
        }
        return ['data' => $user->jsonSerialize(), 'code' => 200];
    }

    public function store(array $request): array
    {
        $command = new CreateUserCommand(
            $request['name'] ?? '',
            $request['email'] ?? '',
            $request['role'] ?? 'user'
        );
        if (!$command->validate()) {
            return ['error' => 'Validation failed', 'code' => 422];
        }
        $user = $this->userService->findByEmail($command->getEmail());
        if ($user !== null) {
            return ['error' => 'Email already in use', 'code' => 409];
        }
        return ['data' => [], 'code' => 201];
    }

    public function suspend(int $id): array
    {
        try {
            $this->userService->suspendUser($id, 'Admin action');
            return ['code' => 200];
        } catch (\RuntimeException $e) {
            return ['error' => $e->getMessage(), 'code' => 404];
        }
    }

    public function promote(int $id): array
    {
        $user = $this->userService->promoteToAdmin($id);
        return ['data' => $user->jsonSerialize(), 'code' => 200];
    }

    public function withOrders(int $id): array
    {
        return $this->userService->getUserWithOrders($id);
    }
}
