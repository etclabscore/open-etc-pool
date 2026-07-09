<script lang="ts">
  let { size = 28 }: { size?: number } = $props();
</script>

<!-- Faceted Ethereum-Classic diamond in the pool's green. -->
<svg width={size} height={size} viewBox="0 0 32 32" aria-hidden="true" class="brand-logo">
  <polygon points="16,2 6,16 16,21" fill="#61bd81" />
  <polygon points="16,2 26,16 16,21" fill="#4faf72" />
  <polygon points="16,23 6,17 16,30" fill="#61bd81" />
  <polygon points="16,23 26,17 16,30" fill="#4faf72" />
</svg>
