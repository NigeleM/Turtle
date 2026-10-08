best = 1e18; check = 0; n = 200
3.times do
  t = Process.clock_gettime(Process::CLOCK_MONOTONIC)
  a = Array.new(n) { Array.new(n, 0.0) }; b = Array.new(n) { Array.new(n, 0) }; rhs = []
  n.times do |i|
    n.times { |j| v = i * 7 + j * 13; a[i][j] = (v % 10).to_f; b[i][j] = v % 7 }
    a[i][i] = (n * 10 + i % 10).to_f; rhs << i % 7 + 1
  end
  bt = b.transpose
  total = 0.0
  a.each { |row| bt.each { |col| s = 0.0; n.times { |k| s += row[k] * col[k] }; total += s } }
  m = a.each_with_index.map { |row, i| row + [rhs[i].to_f] }
  n.times do |k|
    p = (k...n).max_by { |r2| m[r2][k].abs }
    m[k], m[p] = m[p], m[k]
    ((k + 1)...n).each do |r2|
      f = m[r2][k] / m[k][k]
      next if f == 0
      rk = m[k]; rr = m[r2]
      (k..n).each { |j| rr[j] -= f * rk[j] }
    end
  end
  x = Array.new(n, 0.0)
  (n - 1).downto(0) { |i| s = m[i][n]; ((i + 1)...n).each { |j| s -= m[i][j] * x[j] }; x[i] = s / m[i][i] }
  check = (total + (x.sum * 1000000).round).to_i
  best = [best, (Process.clock_gettime(Process::CLOCK_MONOTONIC) - t) * 1000].min
end
puts "result=#{check} ms=#{'%.1f' % best}"
