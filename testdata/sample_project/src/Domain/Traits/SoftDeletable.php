<?php

namespace App\Domain\Traits;

trait SoftDeletable
{
    use Timestampable;

    private bool $deleted = false;
    private ?\DateTime $deletedAt = null;

    public function softDelete(): void
    {
        $this->deleted = true;
        $this->deletedAt = new \DateTime();
        $this->touch();
    }

    public function restore(): void
    {
        $this->deleted = false;
        $this->deletedAt = null;
    }

    public function isDeleted(): bool
    {
        return $this->deleted;
    }
}
