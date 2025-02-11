<script lang="ts">
  import { V1DeploymentStatus } from "@rilldata/web-admin/client";
  import { getDashboardsForProject } from "@rilldata/web-admin/components/projects/dashboards";
  import { invalidateProjectQueries } from "@rilldata/web-admin/components/projects/invalidations";
  import { useProject } from "@rilldata/web-admin/components/projects/use-project";
  import CancelCircle from "@rilldata/web-common/components/icons/CancelCircle.svelte";
  import CheckCircle from "@rilldata/web-common/components/icons/CheckCircle.svelte";
  import Spacer from "@rilldata/web-common/components/icons/Spacer.svelte";
  import Spinner from "@rilldata/web-common/features/entity-management/Spinner.svelte";
  import { EntityStatus } from "@rilldata/web-common/features/entity-management/types";
  import { useQueryClient } from "@tanstack/svelte-query";
  import type { SvelteComponent } from "svelte";

  export let organization: string;
  export let project: string;
  export let iconOnly = false;

  $: proj = useProject(organization, project);
  let deploymentStatus: V1DeploymentStatus;
  $: currentStatusDisplay =
    !!deploymentStatus && statusDisplays[deploymentStatus];

  const queryClient = useQueryClient();

  $: if ($proj.data?.prodDeployment?.status) {
    const prevStatus = deploymentStatus;

    deploymentStatus = $proj.data?.prodDeployment?.status;

    if (
      prevStatus !== V1DeploymentStatus.DEPLOYMENT_STATUS_OK &&
      deploymentStatus === V1DeploymentStatus.DEPLOYMENT_STATUS_OK
    ) {
      getDashboardsAndInvalidate();
    }
  }

  async function getDashboardsAndInvalidate() {
    return invalidateProjectQueries(
      queryClient,
      await getDashboardsForProject($proj.data)
    );
  }

  type StatusDisplay = {
    icon: typeof SvelteComponent;
    iconProps?: {
      [key: string]: unknown;
    };
    text?: string;
    textClass?: string;
    wrapperClass?: string;
  };

  const statusDisplays: Record<V1DeploymentStatus, StatusDisplay> = {
    [V1DeploymentStatus.DEPLOYMENT_STATUS_OK]: {
      icon: CheckCircle,
      iconProps: { className: "text-blue-600 hover:text-blue-500" },
      text: "ready",
      textClass: "text-blue-600",
      wrapperClass: "bg-blue-50 border-blue-300",
    },
    [V1DeploymentStatus.DEPLOYMENT_STATUS_PENDING]: {
      icon: Spinner,
      iconProps: {
        className: "text-purple-600 hover:text-purple-500",
        status: EntityStatus.Running,
      },
      text: "syncing",
      textClass: "text-purple-600",
      wrapperClass: "bg-purple-50 border-purple-300",
    },
    [V1DeploymentStatus.DEPLOYMENT_STATUS_RECONCILING]: {
      icon: Spinner,
      iconProps: {
        className: "text-purple-600 hover:text-purple-500",
        status: EntityStatus.Running,
      },
      text: "syncing",
      textClass: "text-purple-600",
      wrapperClass: "bg-purple-50 border-purple-300",
    },
    [V1DeploymentStatus.DEPLOYMENT_STATUS_ERROR]: {
      icon: CancelCircle,
      iconProps: { className: "text-red-600 hover:text-red-500" },
      text: "error",
      textClass: "text-red-600",
      wrapperClass: "bg-red-50 border-red-300",
    },
    [V1DeploymentStatus.DEPLOYMENT_STATUS_UNSPECIFIED]: {
      icon: Spacer,
    },
  };
</script>

{#if deploymentStatus && deploymentStatus !== V1DeploymentStatus.DEPLOYMENT_STATUS_UNSPECIFIED}
  {#if iconOnly}
    <svelte:component
      this={currentStatusDisplay.icon}
      {...currentStatusDisplay.iconProps}
    />
  {:else}
    <div
      class="flex space-x-1 items-center px-2 border rounded rounded-[20px] w-fit {currentStatusDisplay.wrapperClass} {iconOnly &&
        'hidden'}"
    >
      <svelte:component
        this={currentStatusDisplay.icon}
        {...currentStatusDisplay.iconProps}
      />
      <span class={currentStatusDisplay.textClass}
        >{currentStatusDisplay.text}</span
      >
    </div>
  {/if}
{:else}
  <!-- Avoid layout shift for the iconOnly instance on the homepage -->
  <Spacer />
{/if}
