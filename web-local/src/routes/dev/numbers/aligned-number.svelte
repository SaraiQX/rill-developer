<script lang="ts">
  import type { RichFormatNumber } from "./number-to-string-formatters";

  export let containerWidth = 81;

  export let richNum: RichFormatNumber;
  export let alignSuffix = false;
  export let suffixPadding = 0;
  export let showMagSuffixForZero = false;

  export let lowerCaseEForEng = false;
  export let alignDecimalPoints = false;
  export let zeroHandling: "noSpecial" | "exactZero" | "zeroDot" = "noSpecial";

  $: int = richNum.splitStr.int;
  $: frac = richNum.splitStr.frac;
  $: suffix = richNum.splitStr.suffix;

  // FINALIZE CHARACTERS TO BE DISPLAYED
  let suffixFinal;
  $: {
    // console.log({ lowerCaseEForEng });
    suffixFinal = suffix;
    if (lowerCaseEForEng) suffixFinal = suffixFinal.replace("E", "e");

    if (richNum.number === 0 && !showMagSuffixForZero) suffixFinal = "";
  }

  let decimalPoint: "" | ".";
  $: {
    decimalPoint = richNum.splitStr.dot;
    if (richNum.number === 0) {
      if (zeroHandling === "exactZero") {
        decimalPoint = "";
        frac = "";
      } else if (zeroHandling === "zeroDot") {
        decimalPoint = ".";
        frac = "";
      }
    }
  }

  $: suffixPadFinal = richNum.maxPxWidth.suffix > 0 ? suffixPadding : 0;

  $: intPx = richNum.maxPxWidth.int;
  $: dotPx = richNum.maxPxWidth.dot;
  $: fracPx = richNum.maxPxWidth.frac;
  $: suffixPx = richNum.maxPxWidth.suffix + suffixPadFinal;

  // $: containerWidth = `${intPx + dotPx + fracPx + suffixPx}px`;

  $: fracAndSuffixWidth = `${dotPx + fracPx + suffixPadFinal + suffixPx}px`;

  $: logProps = () => {
    console.log({ ...richNum, lowerCaseEForEng });
  };
</script>

<!-- svelte-ignore a11y-click-events-have-key-events -->
<div
  on:click={logProps}
  class="number-container"
  style="width: {containerWidth}px;"
>
  {#if !alignDecimalPoints}
    {#if alignSuffix}
      {int}{decimalPoint}{frac}
      <div
        class="number-suff"
        style="width: {suffixPx}px; padding-left: {suffixPadFinal}px"
      >
        {suffixFinal}
      </div>
    {:else}
      {int}{decimalPoint}{frac}
      {#if suffixFinal != ""}
        <div class="number-suff" style="padding-left: {suffixPadFinal}px">
          {suffixFinal}
        </div>
      {/if}
    {/if}
  {:else}
    <div class="number-whole" style="width: {intPx}px;">
      {int}
    </div>
    {#if alignSuffix}
      <div class="number-frac" style="width: {dotPx + fracPx}px;">
        {decimalPoint}{frac}
      </div>

      <div
        class="number-suff"
        style="width: {suffixPx}px; padding-left: {suffixPadFinal}px"
      >
        {suffixFinal}
      </div>
    {:else}
      <div class="number-frac-and-suff" style="width: {fracAndSuffixWidth};">
        {decimalPoint}{frac}<span
          class="number-suff"
          style="padding-left: {suffixPadFinal}px"
        >
          {suffixFinal}
        </span>
      </div>
    {/if}
  {/if}
</div>

<style>
  div.number-container {
    display: flex;
    flex-direction: row;
    justify-content: flex-end;
    flex-wrap: nowrap;
    white-space: nowrap;
    overflow: hidden;
    position: relative;
  }

  div.number-whole {
    text-align: right;
  }
  div.number-frac {
    text-align: left;
  }

  div.number-suff {
    text-align: left;
  }

  div.number-frac-and-suff {
    text-align: left;
  }
</style>
