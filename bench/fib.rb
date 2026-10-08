def fib(n) n < 2 ? n : fib(n - 1) + fib(n - 2) end
best = nil; result = nil
3.times { t = Process.clock_gettime(Process::CLOCK_MONOTONIC); result = fib(27); took = (Process.clock_gettime(Process::CLOCK_MONOTONIC) - t) * 1000; best = took if best.nil? || took < best }
puts "result=#{result} ms=#{'%.1f' % best}"
