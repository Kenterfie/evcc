<template>
	<div class="d-flex justify-content-between mb-3 align-items-center" data-testid="vehicle-title">
		<h4 class="d-flex align-items-center m-0 flex-grow-1 overflow-hidden">
			<div
				v-if="iconType === 'refresh'"
				ref="refresh"
				class="me-2 flex-shrink-0 spin"
				:title="$t('main.vehicle.detectionActive')"
				data-bs-toggle="tooltip"
			>
				<Sync />
			</div>
			<VehicleIcon
				v-else-if="iconType === 'vehicle'"
				:name="icon"
				class="me-2 flex-shrink-0"
			/>
			<shopicon-regular-cablecharge
				v-else
				class="me-2 flex-shrink-0"
			></shopicon-regular-cablecharge>
			<VehicleOptions
				v-if="showOptions"
				v-bind="vehicleOptionsProps"
				:id="id"
				class="options"
				:selected="vehicleName"
				@change-vehicle="changeVehicle"
				@remove-vehicle="removeVehicle"
			>
				<div class="flex-grow-1 vehicle-name" data-testid="vehicle-name">
					<span class="d-inline-block text-truncate vehicle-title">{{ name }}</span>
					<span v-if="formattedLastUpdate" class="last-update">{{ formattedLastUpdate }}</span>
				</div>
			</VehicleOptions>
			<div v-else class="flex-grow-1 vehicle-name" data-testid="vehicle-name">
				<span class="d-inline-block text-truncate vehicle-title">{{ name }}</span>
				<span v-if="formattedLastUpdate" class="last-update">{{ formattedLastUpdate }}</span>
			</div>
			<button
				v-if="vehicleNotReachable"
				ref="notReachable"
				class="ms-2 btn-neutral"
				:title="`${$t('main.vehicle.notReachable')} - ${$t('main.vehicle.retryConnection')}`"
				type="button"
				data-testid="vehicle-not-reachable-icon"
				@click="retryVehicleConnection"
				@contextmenu.prevent="openHelpModal"
			>
				<CloudOffline class="evcc-gray" />
			</button>
		</h4>
	</div>
</template>

<script lang="ts">
import "@h2d2/shopicons/es/regular/cablecharge";
import Tooltip from "bootstrap/js/dist/tooltip";
import Modal from "bootstrap/js/dist/modal";
import VehicleIcon from "../VehicleIcon";
import Options from "./Options.vue";
import CloudOffline from "../MaterialIcon/CloudOffline.vue";
import Sync from "../MaterialIcon/Sync.vue";
import collector from "@/mixins/collector";
import formatter from "@/mixins/formatter";
import { defineComponent, type PropType } from "vue";
import type { SelectOption, Vehicle } from "@/types/evcc";

export default defineComponent({
	name: "VehicleTitle",
	components: { VehicleOptions: Options, VehicleIcon, Sync, CloudOffline },
	mixins: [collector, formatter],
	props: {
		connected: Boolean,
		id: [String, Number],
		vehicleDetectionActive: Boolean,
		vehicleNotReachable: Boolean,
		icon: String,
		vehicleName: String,
		vehicles: { type: Array as PropType<Vehicle[]>, default: () => [] },
		title: String,
		lastUpdate: { type: String, default: null },
	},
	emits: ["change-vehicle", "remove-vehicle"],
	data() {
		return {
			refreshTooltip: null as Tooltip | null,
			notReachableTooltip: null as Tooltip | null,
		};
	},
	computed: {
		iconType() {
			if (this.vehicleDetectionActive) {
				return "refresh";
			}
			if (this.connected) {
				return "vehicle";
			}
			return null;
		},
		name() {
			if (this.title) {
				return this.title;
			}
			if (this.connected) {
				return this.$t("main.vehicle.unknown");
			}
			return this.$t("main.vehicle.none");
		},
		vehicleOptions(): SelectOption<string>[] {
			return this.vehicles.map((v) => ({
				name: v.name,
				value: v.title,
			}));
		},
		vehicleKnown() {
			return !!this.vehicleName;
		},
		showOptions() {
			return this.vehicleKnown || this.vehicles.length;
		},
		vehicleOptionsProps() {
			return this.collectProps(Options);
		},
		formattedLastUpdate() {
			if (!this.lastUpdate) {
				return "";
			}
			const date = new Date(this.lastUpdate);
			if (isNaN(date.getTime())) {
				return "";
			}
			const now = new Date();
			const diffMs = now.getTime() - date.getTime();
			const diffMins = Math.floor(diffMs / 60000);

			if (diffMins < 1) {
				return this.$t("main.vehicle.updateNow") as string;
			}
			if (diffMins < 60) {
				return this.$t("main.vehicle.updateMinutesAgo", { minutes: diffMins }) as string;
			}
			if (diffMins < 1440) {
				const hours = Math.floor(diffMins / 60);
				return this.$t("main.vehicle.updateHoursAgo", { hours }) as string;
			}
			return this.fmtDayMonth(date);
		},
	},
	watch: {
		iconType() {
			this.initTooltip();
		},
	},
	mounted() {
		this.initTooltip();
	},
	methods: {
		changeVehicle(name: string) {
			this.$emit("change-vehicle", name);
		},
		removeVehicle() {
			this.$emit("remove-vehicle");
		},
		async retryVehicleConnection() {
			try {
				const url = `/api/loadpoints/${this.id}/vehicle`;
				const response = await fetch(url, { method: "PATCH" });
				if (!response.ok) {
					console.error("Failed to retry vehicle connection:", response.statusText);
				}
			} catch (error) {
				console.error("Error retrying vehicle connection:", error);
			}
		},
		initTooltip() {
			this.$nextTick(() => {
				this.refreshTooltip?.dispose();
				this.notReachableTooltip?.dispose();
				if (this.$refs["refresh"]) {
					this.refreshTooltip = new Tooltip(this.$refs["refresh"]);
				}
				if (this.$refs["notReachable"]) {
					this.notReachableTooltip = new Tooltip(this.$refs["notReachable"]);
				}
			});
		},
		openHelpModal() {
			const modal = Modal.getOrCreateInstance(
				document.getElementById("helpModal") as HTMLElement
			);
			modal.show();
			this.initTooltip();
		},
	},
});
</script>

<style scoped>
.vehicle-name .vehicle-title {
	text-decoration: underline;
	text-decoration-color: var(--evcc-gray);
	max-width: 100%;
}
.vehicle-name .last-update {
	display: block;
	font-size: 0.75rem;
	font-weight: normal;
	color: var(--evcc-gray);
	margin-top: 0.25rem;
	text-decoration: none;
}
.spin {
	animation: rotation 1s infinite cubic-bezier(0.37, 0, 0.63, 1);
}
@keyframes rotation {
	from {
		transform: rotate(0deg);
	}
	to {
		transform: rotate(-360deg);
	}
}
</style>
