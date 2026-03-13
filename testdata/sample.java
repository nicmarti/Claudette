import java.util.List;
import java.util.ArrayList;

interface Speakable {
    String speak();
}

class Animal implements Speakable {
    protected String name;

    Animal(String name) {
        this.name = name;
    }

    public String speak() {
        return "";
    }
}

class Dog extends Animal {
    Dog(String name) {
        super(name);
    }

    @Override
    public String speak() {
        return name + " says Woof!";
    }
}

enum AnimalType {
    DOG, CAT, BIRD
}

class AnimalHelper {
    public static String greet(Speakable animal) {
        return animal.speak();
    }

    public static List<Animal> createDogs(String... names) {
        List<Animal> dogs = new ArrayList<>();
        for (String name : names) {
            dogs.add(new Dog(name));
        }
        return dogs;
    }

    @Test
    public void testDogSpeak() {
        Dog dog = new Dog("Rex");
        assert dog.speak().equals("Rex says Woof!");
    }
}
