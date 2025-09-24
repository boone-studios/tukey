<?php

function format_phone($phone) {
    return preg_replace('/[^\d]/', '', $phone);
}

function format_address($address) {
    return trim($address);
}
