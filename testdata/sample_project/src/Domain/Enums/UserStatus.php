<?php

namespace App\Domain\Enums;

enum UserStatus: string
{
    case Active = 'active';
    case Inactive = 'inactive';
    case Suspended = 'suspended';
    case PendingVerification = 'pending_verification';

    public function isActive(): bool
    {
        return $this === self::Active;
    }

    public function canLogin(): bool
    {
        return $this === self::Active;
    }

    public function label(): string
    {
        return ucwords(str_replace('_', ' ', $this->value));
    }
}
