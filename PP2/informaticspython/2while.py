a = int(input())
i = a - 1
ans = a
while i != 1:
    if a % i == 0:
        ans = i
        i =i - 1
    else:
        i = i - 1
print(ans)