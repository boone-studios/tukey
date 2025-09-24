<?php

namespace App\Http\Controllers;

class UserController {
    public function store($request) {
        $phone = format_phone($request->phone);
        return response()->json(['phone' => $phone]);
    }

    public function update($request) {
        $phone = format_phone($request->phone);
        return response()->json(['phone' => $phone]);
    }
}
