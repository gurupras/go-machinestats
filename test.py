from collections import OrderedDict
import time
import json

data = [
    '''cpu  16237003 488024 7674741 943706235 1071665 0 1072139 0 0 0
cpu0 639241 21228 326322 39345419 43681 0 685277 0 0 0
cpu1 611316 23634 309018 39395345 46618 0 191330 0 0 0
cpu2 660324 20703 308762 39355439 47733 0 78288 0 0 0
cpu3 681919 21493 325023 39313594 45523 0 43927 0 0 0
cpu4 648886 20282 324626 39356106 40980 0 15617 0 0 0
cpu5 642872 20554 321832 39351907 44908 0 10070 0 0 0
cpu6 688975 21220 377022 39251627 51747 0 4367 0 0 0
cpu7 669572 21184 334274 39326310 41296 0 4828 0 0 0''',
    '''cpu  16334594 488336 7714010 947760072 1075645 0 1077789 0 0 0
cpu0 643042 21308 328043 39514523 43789 0 689301 0 0 0
cpu1 615338 23684 310567 39564404 46821 0 192208 0 0 0
cpu2 664600 20703 310336 39524229 47913 0 78571 0 0 0
cpu3 686230 21546 326600 39482489 45603 0 44100 0 0 0
cpu4 653093 20282 326248 39524930 41199 0 15662 0 0 0
cpu5 646487 20560 323492 39521227 45076 0 10102 0 0 0
cpu6 694338 21222 379024 39418809 51932 0 4390 0 0 0
cpu7 673592 21188 336084 39495119 41442 0 4846 0 0 0'''
]
#print 'Processing /proc/stat ...'
#for i in range(3):
#    with open('/proc/stat') as f:
#        content = f.read()
#        data.append(content)
#    time.sleep(1)

cpuData = {}

results = []
for entry in data:
    lines = [x for x in entry.split('\n') if x.strip() != '']
    result = OrderedDict()
    for line in lines:
        tokens = [x for x in line.split(' ') if x != '']
        label = tokens[0]
        if not label.startswith('cpu'):
            continue
        user, nice, system, idle, ioWait, irq, softIRQ, steal, guest, guestNice = [int(x) / float(100) for x in tokens[1:]]
        # There is some double-counting here
        userTime = user - guest
        niceTime = nice - guestNice
        idleAllTime = idle + ioWait
        systemAllTime = system + irq + softIRQ
        virtAllTime = guest + guestNice
        totalAllTime = userTime + niceTime + systemAllTime + idleAllTime + steal + virtAllTime

        if not cpuData.get(label, None):
            cpuData[label] = {
                'total': totalAllTime,
                'idle': idleAllTime,
            }
            continue
        info = cpuData[label]
        prevTotal = info['total']
        prevIdle = info['idle']

        deltaTotal = totalAllTime - prevTotal
        deltaIdle = idleAllTime - prevIdle

        busy = float(deltaTotal - deltaIdle) / float(deltaTotal)
        result[label] = busy
    if len(result) > 0:
        results.append(result.values())

print json.dumps(results, indent=2)
