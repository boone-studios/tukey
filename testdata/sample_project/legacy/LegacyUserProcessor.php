<?php

// This class is no longer referenced anywhere. Replaced by App\Application\UserService in v2.0.

namespace App\Legacy;

class LegacyUserProcessor
{
    private array $users = [];
    private static int $processedCount = 0;

    public function process(array $userData): array
    {
        $result = [];
        foreach ($userData as $user) {
            $processed = $this->normalizeUser($user);
            $result[] = $processed;
            self::$processedCount++;
        }
        return $result;
    }

    private function normalizeUser(array $user): array
    {
        return [
            'name'  => ucwords(strtolower($user['name'] ?? '')),
            'email' => strtolower($user['email'] ?? ''),
            'phone' => preg_replace('/[^\d+]/', '', $user['phone'] ?? ''),
        ];
    }

    public function validateEmail(string $email): bool
    {
        return filter_var($email, FILTER_VALIDATE_EMAIL) !== false;
    }

    public static function getProcessedCount(): int
    {
        return self::$processedCount;
    }

    public function batchProcess(array $batches): array
    {
        $results = [];
        foreach ($batches as $batch) {
            $results = array_merge($results, $this->process($batch));
        }
        return $results;
    }
}
