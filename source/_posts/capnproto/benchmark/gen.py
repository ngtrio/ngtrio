from random import Random

elems = 'abcdefghijklmnopqrstuvwxyz0123456789'

if __name__ == '__main__':
    last = len(elems) - 1
    rand = Random()
    with open("data", 'w') as data:
        for i in range(1000000):
            randomLength = rand.randint(5, 10)
            s = ''
            for j in range(randomLength):
                idx = rand.randint(0, last)
                s += elems[idx]
            n = rand.randint(1, 1000)
            data.write(s + '\n')
            data.write(str(n) + '\n')
