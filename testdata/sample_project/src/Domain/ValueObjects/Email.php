<?php

namespace App\Domain\ValueObjects;

final class Email
{
    private string $address;

    public function __construct(string $address)
    {
        $this->address = strtolower($address);
    }

    public static function fromString(string $address): self
    {
        return new self($address);
    }

    public function getValue(): string
    {
        return $this->address;
    }

    public function getDomain(): string
    {
        return substr($this->address, strpos($this->address, '@') + 1);
    }

    public function equals(Email $other): bool
    {
        return $this->address === $other->address;
    }

    public function __toString(): string
    {
        return $this->address;
    }
}
