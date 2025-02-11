// Create types and then present an appropriate string
// Current dash persion has `prefix` key in JSON to add currecny etc.
// We can provide a dropdown option in the table?? or regex??

import { humanizedFormatterFactory } from "@rilldata/web-common/lib/number-formatting/humanizer";
import {
  FormatterFactoryOptions,
  NumberKind,
} from "@rilldata/web-common/lib/number-formatting/humanizer-types";
import { PerRangeFormatter } from "@rilldata/web-common/lib/number-formatting/strategies/per-range";
import type { LeaderboardValue } from "./dashboard-stores";

const shortHandSymbols = ["Q", "T", "B", "M", "k", "none"] as const;
export type ShortHandSymbols = typeof shortHandSymbols[number];

interface HumanizeOptions {
  scale?: ShortHandSymbols;
  excludeDecimalZeros?: boolean;
  columnName?: string;
}

type formatterOptions = Intl.NumberFormatOptions & HumanizeOptions;

const shortHandMap = {
  Q: 1.0e15,
  T: 1.0e12,
  B: 1.0e9,
  M: 1.0e6,
  k: 1.0e3,
  none: 1,
};

export enum NicelyFormattedTypes {
  HUMANIZE = "humanize",
  NONE = "none",
  CURRENCY = "currency_usd",
  PERCENTAGE = "percentage",
}

interface ColFormatSpec {
  columnName: string;
  formatPreset: NicelyFormattedTypes;
}

export const nicelyFormattedTypesSelectorOptions = [
  { value: NicelyFormattedTypes.HUMANIZE, label: "Humanize" },
  {
    value: NicelyFormattedTypes.NONE,
    label: "No formatting",
  },
  {
    value: NicelyFormattedTypes.CURRENCY,
    label: "Currency (USD)",
  },
  {
    value: NicelyFormattedTypes.PERCENTAGE,
    label: "Percentage",
  },
];

function getScaleForValue(value: number): ShortHandSymbols {
  return Math.abs(value) >= 1.0e15
    ? "Q"
    : Math.abs(value) >= 1.0e12
    ? "T"
    : Math.abs(value) >= 1.0e9
    ? "B"
    : Math.abs(value) >= 1.0e6
    ? "M"
    : Math.abs(value) >= 1.0e3
    ? "k"
    : "none";
}

function determineScaleForValues(values: number[]): ShortHandSymbols {
  let numberValues = values;
  const nullIndex = values.indexOf(null);
  if (nullIndex !== -1) {
    numberValues = values.slice(0, nullIndex);
  }

  // Convert negative numbers to absolute
  numberValues = numberValues.map((v) => Math.abs(v)).sort((a, b) => b - a);

  const half = Math.floor(numberValues.length / 2);
  let median: number;
  if (numberValues.length % 2) median = numberValues[half];
  else median = (numberValues[half - 1] + numberValues[half]) / 2.0;

  let scaleForMax = getScaleForValue(numberValues[0]);
  while (scaleForMax != shortHandSymbols[shortHandSymbols.length - 1]) {
    const medianShorthand = (
      Math.abs(median) / shortHandMap[scaleForMax]
    ).toFixed(1);

    const numDigitsInMedian = medianShorthand.toString().split(".")[0].length;
    if (numDigitsInMedian >= 1) {
      return scaleForMax;
    } else {
      scaleForMax = shortHandSymbols[shortHandSymbols.indexOf(scaleForMax) + 1];
    }
  }
  return scaleForMax;
}

export function humanizeGroupValues(
  values: Array<Record<string, number | string>>,
  type: NicelyFormattedTypes,
  options?: formatterOptions
) {
  const valueKey = options.columnName ? options.columnName : "value";
  let numValues = values.map((v) => v[valueKey]);

  const areAllNumbers = numValues.some((e) => typeof e === "number");
  if (!areAllNumbers) return values;

  numValues = (numValues as number[]).sort((a, b) => b - a);
  const formattedValues = humanizeGroupValuesUtil2(
    numValues as number[],
    type,
    options
  );

  const formattedValueKey = "__formatted_" + valueKey;
  const humanizedValues = values.map((v) => {
    const index = numValues.indexOf(v[valueKey]);
    return { ...v, [formattedValueKey]: formattedValues[index] };
  });

  return humanizedValues;
}

export function humanizeGroupByColumns(
  values: Array<Record<string, number | string>>,
  columnFormatSpec: ColFormatSpec[]
) {
  return columnFormatSpec.reduce((valuesObj, column) => {
    return humanizeGroupValues(
      valuesObj,
      column.formatPreset || NicelyFormattedTypes.HUMANIZE,
      {
        columnName: column.columnName,
      }
    );
  }, values);
}

export function getScaleForLeaderboard(
  leaderboard: Map<string, Array<LeaderboardValue>>
) {
  if (!leaderboard) return "none";

  const numValues = [...leaderboard.values()]
    // use the first five dimensions as the sample
    .slice(0, 5)
    // Take only first 7 values which are shown as input
    .map((values) => values.slice(0, 7))
    .flat()
    .map((values) => values.value);

  const areAllNumbers = numValues.every((e) => typeof e === "number");
  if (!areAllNumbers) return "none";

  const sortedValues = numValues.sort((a, b) => b - a);

  return determineScaleForValues(sortedValues);
}

// NOTE: the following are adapters that I think fit the API
// used by the existing humanizer, but I'm not sure of the
// exact details, nor am I totally confident about the options
// passed in at all the relevant call sites, so I've added
// thes adapters rather than just pave over the existing functions.
// This really needs to be reviewed by Dhiraj, at which point we
// can deprecate any left over code that is no longer needed.

export const nicelyFormattedTypesToNumberKind = (
  type: NicelyFormattedTypes | string
) => {
  switch (type) {
    case NicelyFormattedTypes.CURRENCY:
      return NumberKind.DOLLAR;

    case NicelyFormattedTypes.PERCENTAGE:
      return NumberKind.PERCENT;

    default:
      // captures:
      // NicelyFormattedTypes.NONE
      // NicelyFormattedTypes.HUMANIZE
      return NumberKind.ANY;
  }
};

export function humanizeDataType(
  value: unknown,
  type: NicelyFormattedTypes,
  options?: FormatterFactoryOptions
): string {
  if (value === undefined || value === null) return "";
  if (typeof value != "number") return value.toString();

  const numberKind = nicelyFormattedTypesToNumberKind(type);

  let innerOptions: FormatterFactoryOptions = options;
  if (type === NicelyFormattedTypes.NONE) {
    innerOptions = {
      strategy: "none",
      numberKind,
      padWithInsignificantZeros: false,
    };
  } else if (options === undefined) {
    innerOptions = {
      strategy: "default",
      numberKind,
    };
  } else {
    innerOptions = {
      strategy: "default",
      ...options,
      numberKind,
    };
  }
  return humanizedFormatterFactory([value], innerOptions).stringFormat(value);
}

/** This function is used primarily in the leaderboard and the detail tables. */
function humanizeGroupValuesUtil2(
  values: number[],
  type: NicelyFormattedTypes,
  _options?: formatterOptions
) {
  if (!values.length) return values;
  if (type == NicelyFormattedTypes.NONE) return values;

  const numberKind = nicelyFormattedTypesToNumberKind(type);

  const innerOptions: FormatterFactoryOptions = {
    strategy: "default",
    numberKind,
  };

  const formatter = humanizedFormatterFactory(values, innerOptions);

  return values.map((v) => {
    if (v === null) return "∅";
    else return formatter.stringFormat(v);
  });
}

/** formatter for the comparison percentage differences */
export function formatMeasurePercentageDifference(
  value,
  method = "partsFormat"
) {
  if (Math.abs(value * 100) < 1 && value !== 0) {
    return method === "partsFormat"
      ? { percent: "%", neg: "", int: "<1" }
      : "<1%";
  } else if (value === 0) {
    return method === "partsFormat" ? { percent: "%", neg: "", int: 0 } : "0%";
  }
  const factory = new PerRangeFormatter([], {
    strategy: "perRange",
    rangeSpecs: [
      {
        minMag: -2,
        supMag: 3,
        maxDigitsRight: 1,
        baseMagnitude: 0,
        padWithInsignificantZeros: false,
      },
    ],
    defaultMaxDigitsRight: 0,
    numberKind: NumberKind.PERCENT,
  });

  return factory[method](value);
}
