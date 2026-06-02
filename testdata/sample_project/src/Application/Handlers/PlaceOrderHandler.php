<?php

namespace App\Application\Handlers;

use App\Application\Commands\PlaceOrderCommand;
use App\Application\Events\OrderPlaced;
use App\Domain\{Order, OrderItem, User};
use App\Domain\OrderRepository;
use App\Domain\UserRepository;
use App\Domain\ValueObjects\Money;

class PlaceOrderHandler
{
    private OrderRepository $orderRepository;
    private UserRepository $userRepository;

    public function __construct(OrderRepository $orderRepository, UserRepository $userRepository)
    {
        $this->orderRepository = $orderRepository;
        $this->userRepository = $userRepository;
    }

    public function handle(PlaceOrderCommand $command): Order
    {
        if (!$command->validate()) {
            throw new \InvalidArgumentException("Invalid place order command");
        }

        $user = $this->userRepository->findById($command->getUserId());
        if ($user === null) {
            throw new \RuntimeException("User not found");
        }

        $order = new Order($command->getUserId(), $command->getCurrency());

        foreach ($command->getItems() as $itemData) {
            $price = Money::fromCents($itemData['priceCents'], $command->getCurrency());
            $item = new OrderItem(
                $itemData['productId'],
                $itemData['name'],
                $price,
                $itemData['quantity']
            );
            $order->addItem($item);
        }

        if (!$order->validate()) {
            throw new \DomainException("Order validation failed");
        }

        $this->orderRepository->save($order);

        $event = new OrderPlaced($order, $user);
        return $order;
    }
}
