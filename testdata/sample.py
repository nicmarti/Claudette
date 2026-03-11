import os
from pathlib import Path


class Animal:
    def __init__(self, name: str):
        self.name = name

    def speak(self) -> str:
        return ""


class Dog(Animal):
    def speak(self) -> str:
        return f"{self.name} says Woof!"


def greet(animal: Animal) -> str:
    return animal.speak()


def test_dog_speak():
    dog = Dog("Rex")
    assert dog.speak() == "Rex says Woof!"
