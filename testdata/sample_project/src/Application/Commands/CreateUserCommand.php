<?php

namespace App\Application\Commands;

class CreateUserCommand
{
    private string $name;
    private string $email;
    private string $role;

    public function __construct(string $name, string $email, string $role = 'user')
    {
        $this->name = $name;
        $this->email = $email;
        $this->role = $role;
    }

    public function getName(): string
    {
        return $this->name;
    }

    public function getEmail(): string
    {
        return $this->email;
    }

    public function getRole(): string
    {
        return $this->role;
    }

    public function validate(): bool
    {
        return $this->name !== '' && $this->email !== '';
    }
}
