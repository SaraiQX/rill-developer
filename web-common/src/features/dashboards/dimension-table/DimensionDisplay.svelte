<script lang="ts">
  /**
   * DimensionDisplay.svelte
   * -------------------------
   * Create a table with the selected dimension and measures
   * to be displayed in explore
   */
  import { cancelDashboardQueries } from "@rilldata/web-common/features/dashboards/dashboard-queries";
  import {
    getFilterForDimension,
    useMetaDimension,
    useMetaMeasure,
    useMetaQuery,
    useModelAllTimeRange,
    useModelHasTimeSeries,
  } from "@rilldata/web-common/features/dashboards/selectors";
  import {
    createQueryServiceMetricsViewToplist,
    createQueryServiceMetricsViewTotals,
    MetricsViewDimension,
    MetricsViewFilterCond,
    V1MetricsViewToplistResponse,
  } from "@rilldata/web-common/runtime-client";
  import { useQueryClient } from "@tanstack/svelte-query";
  import { getTimeComparisonParametersForComponent } from "../../../lib/time/comparisons";
  import { DEFAULT_TIME_RANGES } from "../../../lib/time/config";
  import type { TimeComparisonOption } from "../../../lib/time/types";
  import { runtime } from "../../../runtime-client/runtime-store";
  import {
    MetricsExplorerEntity,
    metricsExplorerStore,
  } from "../dashboard-stores";
  import {
    humanizeGroupByColumns,
    NicelyFormattedTypes,
  } from "../humanize-numbers";
  import {
    computeComparisonValues,
    getComparisonProperties,
    getFilterForComparisonTable,
    updateFilterOnSearch,
  } from "./dimension-table-utils";
  import DimensionContainer from "./DimensionContainer.svelte";
  import DimensionHeader from "./DimensionHeader.svelte";
  import DimensionTable from "./DimensionTable.svelte";

  export let metricViewName: string;
  export let dimensionName: string;

  let searchText = "";

  const queryClient = useQueryClient();

  $: instanceId = $runtime.instanceId;

  $: metaQuery = useMetaQuery(instanceId, metricViewName);

  $: dimensionQuery = useMetaDimension(
    instanceId,
    metricViewName,
    dimensionName
  );
  let dimension: MetricsViewDimension;
  $: dimension = $dimensionQuery?.data;

  let metricsExplorer: MetricsExplorerEntity;
  $: metricsExplorer = $metricsExplorerStore.entities[metricViewName];

  $: leaderboardMeasureName = metricsExplorer?.leaderboardMeasureName;
  $: leaderboardMeasureQuery = useMetaMeasure(
    instanceId,
    metricViewName,
    leaderboardMeasureName
  );

  let excludeValues: Array<MetricsViewFilterCond>;
  $: excludeValues = metricsExplorer?.filters.exclude;

  $: excludeMode =
    metricsExplorer?.dimensionFilterExcludeMode.get(dimensionName) ?? false;

  $: filterForDimension = getFilterForDimension(
    metricsExplorer?.filters,
    dimensionName
  );

  $: selectedMeasureNames = metricsExplorer?.selectedMeasureNames;

  let selectedValues: Array<unknown>;
  $: selectedValues =
    (excludeMode
      ? metricsExplorer?.filters.exclude.find((d) => d.name === dimension?.name)
          ?.in
      : metricsExplorer?.filters.include.find((d) => d.name === dimension?.name)
          ?.in) ?? [];

  $: allMeasures = $metaQuery.data?.measures;

  $: sortByColumn = $leaderboardMeasureQuery.data?.name;
  $: sortDirection = sortDirection || "desc";

  $: metricTimeSeries = useModelHasTimeSeries(instanceId, metricViewName);
  $: hasTimeSeries = $metricTimeSeries.data;

  let allTimeRangeQuery;
  let topListQuery;

  $: if (
    sortByColumn &&
    sortDirection &&
    leaderboardMeasureName &&
    metaQuery &&
    $metaQuery.isSuccess &&
    !$metaQuery.isRefetching
  ) {
    let filterSet = updateFilterOnSearch(
      filterForDimension,
      searchText,
      dimension?.name
    );

    let topListParams = {
      dimensionName: dimensionName,
      measureNames: selectedMeasureNames,
      limit: "250",
      offset: "0",
      sort: [
        {
          name: sortByColumn,
          ascending: sortDirection === "asc" ? true : false,
        },
      ],
      filter: filterSet,
    };

    if (hasTimeSeries) {
      topListParams = {
        ...topListParams,
        ...{
          timeStart: metricsExplorer.selectedTimeRange?.start,
          timeEnd: metricsExplorer.selectedTimeRange?.end,
        },
      };
    }

    topListQuery = createQueryServiceMetricsViewToplist(
      instanceId,
      metricViewName,
      topListParams
    );

    allTimeRangeQuery = useModelAllTimeRange(
      $runtime.instanceId,
      $metaQuery.data.model,
      $metaQuery.data.timeDimension
    );
  }

  // the timeRangeName is the key to a selected time range's associated presets.
  $: timeRangeName = metricsExplorer?.selectedTimeRange?.name;

  $: allTimeRange = $allTimeRangeQuery?.data;

  let comparisonTopListQuery;
  let isComparisonRangeAvailable = false;
  let displayComparison = false;

  // create the right compareTopListParams.
  $: if (
    !$topListQuery?.isFetching &&
    hasTimeSeries &&
    timeRangeName !== undefined
  ) {
    const values: V1MetricsViewToplistResponse = $topListQuery?.data?.data;

    const comparisonTimeRange = getTimeComparisonParametersForComponent(
      (metricsExplorer?.selectedComparisonTimeRange
        ?.name as TimeComparisonOption) ||
        (DEFAULT_TIME_RANGES[timeRangeName]
          .defaultComparison as TimeComparisonOption),
      allTimeRange?.start,
      allTimeRange?.end,
      metricsExplorer.selectedTimeRange.start,
      metricsExplorer.selectedTimeRange.end
    );

    const { start, end } = comparisonTimeRange;
    isComparisonRangeAvailable = comparisonTimeRange.isComparisonRangeAvailable;
    displayComparison =
      metricsExplorer?.showComparison && isComparisonRangeAvailable;

    let comparisonFilterSet = getFilterForComparisonTable(
      filterForDimension,
      dimensionName,
      values
    );

    let comparisonParams = {
      dimensionName: dimensionName,
      measureNames: [sortByColumn],
      limit: "250",
      offset: "0",
      sort: [
        {
          name: sortByColumn,
          ascending: sortDirection === "asc" ? true : false,
        },
      ],
      filter: comparisonFilterSet,
    };

    if (hasTimeSeries) {
      comparisonParams = {
        ...comparisonParams,

        ...{
          timeStart: displayComparison ? start : undefined,
          timeEnd: displayComparison ? end : undefined,
        },
      };
    }

    comparisonTopListQuery = createQueryServiceMetricsViewToplist(
      $runtime.instanceId,
      metricViewName,
      comparisonParams
    );
  } else if (!hasTimeSeries) {
    displayComparison = false;
  }

  let totalsQuery;
  $: if (
    metricsExplorer &&
    metaQuery &&
    $metaQuery.isSuccess &&
    !$metaQuery.isRefetching
  ) {
    let totalsQueryParams = { measureNames: selectedMeasureNames };
    if (hasTimeSeries) {
      totalsQueryParams = {
        ...totalsQueryParams,
        ...{
          timeStart: metricsExplorer.selectedTimeRange?.start,
          timeEnd: metricsExplorer.selectedTimeRange?.end,
        },
      };
    }
    totalsQuery = createQueryServiceMetricsViewTotals(
      instanceId,
      metricViewName,
      totalsQueryParams
    );
  }

  let referenceValues = {};
  $: if ($totalsQuery?.data?.data) {
    allMeasures.map((m) => {
      const isSummableMeasure =
        m?.expression.toLowerCase()?.includes("count(") ||
        m?.expression?.toLowerCase()?.includes("sum(");
      if (isSummableMeasure) {
        referenceValues[m.name] = $totalsQuery.data.data?.[m.name];
      }
    });
  }

  let values = [];
  let columns = [];
  let measureNames = [];

  $: if (!$topListQuery?.isFetching && dimension) {
    let columnsMeta = $topListQuery?.data?.meta || [];
    values = $topListQuery?.data?.data ?? [];

    let columnNames: Array<string> = columnsMeta
      .map((c) => c.name)
      .filter((name) => name !== dimension?.name);

    const selectedMeasure = allMeasures.find((m) => m.name === sortByColumn);
    const sortByColumnIndex = columnNames.indexOf(sortByColumn);
    // Add comparison columns if available
    if (displayComparison) {
      columnNames.splice(sortByColumnIndex + 1, 0, `${sortByColumn}_delta`);

      // Only push percentage delta column if selected measure is not a percentage
      if (selectedMeasure?.format != NicelyFormattedTypes.PERCENTAGE) {
        columnNames.splice(
          sortByColumnIndex + 2,
          0,
          `${sortByColumn}_delta_perc`
        );
      }
    }

    // Make dimension the first column
    columnNames.unshift(dimension?.name);
    measureNames = allMeasures.map((m) => m.name);

    columns = columnNames.map((columnName) => {
      if (measureNames.includes(columnName)) {
        // Handle all regular measures
        const measure = allMeasures.find((m) => m.name === columnName);
        return {
          name: columnName,
          type: "INT",
          label: measure?.label || measure?.expression,
          description: measure?.description,
          total: referenceValues[measure.name] || 0,
          enableResize: false,
          format: measure?.format,
        };
      } else if (columnName === dimension?.name) {
        // Handle dimension column
        return {
          name: columnName,
          type: "VARCHAR",
          label: dimension?.label,
          enableResize: true,
        };
      } else {
        // Handle delta and delta_perc
        const comparison = getComparisonProperties(columnName, selectedMeasure);
        return {
          name: columnName,
          type: comparison.type,
          label: comparison.label,
          description: comparison.description,
          enableResize: false,
          format: comparison.format,
        };
      }
    });
  }

  function onSelectItem(event) {
    const label = values[event.detail][dimension?.name];
    cancelDashboardQueries(queryClient, metricViewName);
    metricsExplorerStore.toggleFilter(metricViewName, dimension?.name, label);
  }

  function onSortByColumn(event) {
    const columnName = event.detail;
    if (!measureNames.includes(columnName)) return;

    if (columnName === sortByColumn) {
      sortDirection = sortDirection === "desc" ? "asc" : "desc";
    } else {
      metricsExplorerStore.setLeaderboardMeasureName(
        metricViewName,
        columnName
      );
      sortDirection = "desc";
    }
  }

  $: if ($comparisonTopListQuery?.data && values.length && displayComparison) {
    values = computeComparisonValues(
      $comparisonTopListQuery?.data,
      values,
      dimensionName,
      leaderboardMeasureName
    );
  }

  $: if (values) {
    const measureFormatSpec = columns?.map((column) => {
      return {
        columnName: column.name,
        formatPreset: column.format as NicelyFormattedTypes,
      };
    });
    if (measureFormatSpec) {
      values = humanizeGroupByColumns(values, measureFormatSpec);
    }
  }
</script>

{#if topListQuery}
  <DimensionContainer>
    <DimensionHeader
      {metricViewName}
      {dimensionName}
      {excludeMode}
      isFetching={$topListQuery?.isFetching}
      on:search={(event) => {
        searchText = event.detail;
      }}
    />

    {#if values && columns.length}
      <DimensionTable
        on:select-item={(event) => onSelectItem(event)}
        on:sort={(event) => onSortByColumn(event)}
        dimensionName={dimension?.name}
        {columns}
        {selectedValues}
        rows={values}
        {sortByColumn}
        {excludeMode}
      />
    {/if}
  </DimensionContainer>
{/if}
