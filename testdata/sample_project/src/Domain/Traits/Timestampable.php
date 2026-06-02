<?php

namespace App\Domain\Traits;

trait Timestampable
{
    private ?\DateTime $createdAt = null;
    private ?\DateTime $updatedAt = null;

    public function getCreatedAt(): ?\DateTime
    {
        return $this->createdAt;
    }

    public function setCreatedAt(\DateTime $date): void
    {
        $this->createdAt = $date;
    }

    public function touch(): void
    {
        $this->updatedAt = new \DateTime();
    }
}
