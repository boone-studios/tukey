<?php

// These functions are deprecated and no longer called from production code.

function validate_legacy_email($email) {
    return filter_var($email, FILTER_VALIDATE_EMAIL) !== false;
}

function format_legacy_currency($amount, $currency = 'USD') {
    return sprintf('%s %.2f', $currency, $amount / 100);
}

function get_user_by_email_legacy($email, $db) {
    return $db->query("SELECT * FROM users WHERE email = ?", [$email]);
}

function build_order_total_legacy($items) {
    $total = 0;
    foreach ($items as $item) {
        $total += $item['price'] * $item['qty'];
    }
    return $total;
}
