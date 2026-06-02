<?php

namespace App\Domain;

abstract class Entity
{
    protected ?int $id = null;

    abstract public function getId(): ?int;
    abstract public function validate(): bool;

    public function isNew(): bool
    {
        return $this->id === null;
    }

    public function equals(Entity $other): bool
    {
        if (get_class($this) !== get_class($other)) {
            return false;
        }
        return $this->id !== null && $this->id === $other->id;
    }
}
