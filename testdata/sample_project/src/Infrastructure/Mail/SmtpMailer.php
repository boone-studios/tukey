<?php

namespace App\Infrastructure\Mail;

use App\Domain\User;
use App\Domain\ValueObjects\Email;
use App\Domain\Enums\UserStatus;

class SmtpMailer
{
    const MAX_RETRIES = 3;
    const DEFAULT_FROM = 'noreply@example.com';

    private string $host;
    private int $port;
    private string $username;
    private string $password;
    private string $from;

    public function __construct(string $host, int $port, string $username, string $password)
    {
        $this->host = $host;
        $this->port = $port;
        $this->username = $username;
        $this->password = $password;
        $this->from = self::DEFAULT_FROM;
    }

    public function sendWelcome(User $user): bool
    {
        $to = $user->getEmail()->getValue();
        $subject = 'Welcome!';
        return $this->send($to, $subject, "Hello, welcome aboard!");
    }

    public function sendPasswordReset(Email $email, string $token): bool
    {
        $to = $email->getValue();
        $subject = 'Password Reset';
        return $this->send($to, $subject, "Reset token: {$token}");
    }

    public function sendOrderConfirmation(User $user, int $orderId): bool
    {
        $to = $user->getEmail()->getValue();
        $subject = "Order #{$orderId} Confirmed";
        return $this->send($to, $subject, "Your order has been confirmed.");
    }

    private function send(string $to, string $subject, string $body): bool
    {
        $attempts = 0;
        while ($attempts < self::MAX_RETRIES) {
            $attempts++;
            if ($this->doSend($to, $subject, $body)) {
                return true;
            }
        }
        return false;
    }

    private function doSend(string $to, string $subject, string $body): bool
    {
        return mail($to, $subject, $body, "From: {$this->from}");
    }
}
