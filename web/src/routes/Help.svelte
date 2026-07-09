<script lang="ts">
  import { config } from '../lib/config';

  const mordorFlag = $derived($config.network === 'mordor' ? ' --ecip1099-activation 84' : '');
  const endpoint = $derived($config.stratumHost + ':' + $config.stratumPort);
</script>

<div class="page-header"><h1>How To Mine</h1></div>

<p>
  Point any Etchash miner at the pool's stratum endpoint <code>{endpoint}</code> and set your
  Ethereum Classic wallet address as the username. Replace <code>YOUR_ETC_ADDRESS</code> below with
  it (e.g. <code class="hash">0xda904bc07fd95e39661941b3f6daded1b8a38c71</code>).
</p>

<div class="grid-2">
  <div>
    <h3>lolMiner</h3>
    <p>
      Download <a href="https://github.com/Lolliedieb/lolMiner-releases/releases" target="_blank" rel="noreferrer">lolMiner</a>
      (v1.12+).
    </p>
    <pre>./lolMiner --pool {endpoint} --user YOUR_ETC_ADDRESS -c ETC{mordorFlag} --ethstratum=ETHPROXY</pre>
  </div>

  <div>
    <h3>nanominer <small>(mainnet only)</small></h3>
    <p>
      Download <a href="https://github.com/nanopool/nanominer/releases" target="_blank" rel="noreferrer">nanominer</a>
      (v1.12.0+). Example config:
    </p>
    <pre>[ETC]
wallet = YOUR_ETC_ADDRESS
coin = ETC
pool1 = {endpoint}</pre>
  </div>
</div>

<div class="callout callout-info">
  <p>
    CPU mining is not recommended. By mining on this pool you agree to its Terms of Service; the pool
    is provided as-is, without warranty.
  </p>
</div>
