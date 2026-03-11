import { readFileSync } from 'fs';

interface Speakable {
  speak(): string;
}

class Animal implements Speakable {
  constructor(public name: string) {}

  speak(): string {
    return '';
  }
}

class Dog extends Animal {
  speak(): string {
    return `${this.name} says Woof!`;
  }
}

function greet(animal: Speakable): string {
  return animal.speak();
}

export { Animal, Dog, greet };
