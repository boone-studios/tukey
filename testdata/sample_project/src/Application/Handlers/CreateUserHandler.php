<?php

namespace App\Application\Handlers;

use App\Application\Commands\CreateUserCommand;
use App\Domain\User;
use App\Domain\UserRepository;
use App\Domain\ValueObjects\Email;
use App\Domain\Enums\UserStatus;

class CreateUserHandler
{
    private UserRepository $userRepository;

    public function __construct(UserRepository $userRepository)
    {
        $this->userRepository = $userRepository;
    }

    public function handle(CreateUserCommand $command): User
    {
        if (!$command->validate()) {
            throw new \InvalidArgumentException("Invalid create user command");
        }

        if ($this->userRepository->existsByEmail($command->getEmail())) {
            throw new \RuntimeException("User with that email already exists");
        }

        $email = Email::fromString($command->getEmail());
        $user = new User($command->getName(), $email, UserStatus::PendingVerification);

        if ($command->getRole() === User::ADMIN_ROLE) {
            $user->promote(User::ADMIN_ROLE);
        }

        $this->userRepository->save($user);
        return $user;
    }
}
