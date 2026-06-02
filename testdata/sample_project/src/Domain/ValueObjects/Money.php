<?php

namespace App\Domain\ValueObjects;

final class Money
{
    const ZERO = 0;
    const DEFAULT_CURRENCY = 'USD';

    private int $amount;
    private string $currency;

    private function __construct(int $amount, string $currency)
    {
        $this->amount = $amount;
        $this->currency = $currency;
    }

    public static function fromCents(int $cents, string $currency = self::DEFAULT_CURRENCY): self
    {
        return new self($cents, $currency);
    }

    public static function zero(string $currency = self::DEFAULT_CURRENCY): self
    {
        return new self(self::ZERO, $currency);
    }

    public function add(Money $other): self
    {
        if ($this->currency !== $other->currency) {
            throw new \InvalidArgumentException("Cannot add different currencies");
        }
        return new self($this->amount + $other->amount, $this->currency);
    }

    public function multiply(int $quantity): self
    {
        return new self($this->amount * $quantity, $this->currency);
    }

    public function getAmount(): int
    {
        return $this->amount;
    }

    public function getCurrency(): string
    {
        return $this->currency;
    }

    public function equals(Money $other): bool
    {
        return $this->amount === $other->amount && $this->currency === $other->currency;
    }
}
