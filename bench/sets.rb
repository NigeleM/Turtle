require "set"
best = 1e18; total = 0
3.times { t = Process.clock_gettime(Process::CLOCK_MONOTONIC); a = Set.new; b = Set.new; 200000.times { |i| a << (i * 7) % 300007; b << i * 3 }; hits = 0; 300000.times { |i| hits += 1 if a.include?(i) }; total = hits + (a & b).size + (a | b).size; best = [best, (Process.clock_gettime(Process::CLOCK_MONOTONIC) - t) * 1000].min }
puts "result=#{total} ms=#{'%.1f' % best}"
