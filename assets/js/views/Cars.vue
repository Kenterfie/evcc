<template>
	<div class="container px-4 safe-area-inset">
		<TopHeader :title="$t('config.section.vehicles')" :notifications="notifications" />
		<main>
			<div v-if="cars.length === 0" class="text-muted mt-2">
				{{ $t("main.vehicle.none") }}
			</div>
			<div v-else class="row g-3 pb-3">
				<div v-for="car in cars" :key="car.name" class="col-12 col-md-6 col-xl-4">
					<DeviceCard
						:id="`cars_${car.name}`"
						:name="car.name"
						:title="car.title"
						:no-edit-button="true"
					>
						<template #icon>
							<VehicleIcon :name="car.icon" />
						</template>
						<template #tags>
							<div
								v-if="car.loadpointTitle"
								class="car-loadpoint text-muted small mb-2"
							>
								{{ car.loadpointTitle }}
							</div>
							<DeviceTags :tags="vehicleStatusTags(car)" />
							<div class="car-meta mt-2 pt-2">
								<div class="d-flex gap-2 overflow-hidden text-truncate">
									<div
										class="label overflow-hidden text-truncate flex-shrink-1 flex-grow-1"
									>
										{{ $t("main.vehicleStatus.lastUpdate") }}
									</div>
									<div
										class="value overflow-hidden text-truncate"
										:class="{ 'value--muted': !car.lastUpdate }"
									>
										{{ formattedVehicleLastUpdate(car) || "–" }}
									</div>
								</div>
							</div>
						</template>
					</DeviceCard>
				</div>
			</div>
		</main>
	</div>
</template>

<script lang="ts">
import api from "@/api";
import formatter from "@/mixins/formatter";
import store from "@/store";
import { defineComponent, reactive, watch, type PropType } from "vue";
import DeviceCard from "../components/Config/DeviceCard.vue";
import DeviceTags from "../components/Config/DeviceTags.vue";
import Header from "../components/Top/Header.vue";
import VehicleIcon from "../components/VehicleIcon";
import type { Vehicle, Notification, UiLoadpoint } from "@/types/evcc";

type DeviceTag = { value?: any; error?: boolean; warning?: boolean; muted?: boolean };
type DeviceTagsMap = Record<string, DeviceTag>;

const STATUS_KEYS = [
	"soc",
	"capacity",
	"chargeStatus",
	"vehicleLimitSoc",
	"range",
	"odometer",
] as const;

interface CarState {
	name: string;
	title: string;
	icon: string;
	loadpointTitle: string;
	lastUpdate: string | null;
	soc: number | null;
	capacity: number | null;
	chargeStatus: string | null;
	vehicleLimitSoc: number | null;
	range: number | null;
	odometer: number | null;
}

interface VehicleStatus {
	soc?: number | null;
	range?: number | null;
	odometer?: number | null;
	chargeStatus?: string | null;
	updated?: string | null;
}

export default defineComponent({
	name: "Cars",
	components: {
		TopHeader: Header,
		DeviceCard,
		DeviceTags,
		VehicleIcon,
	},
	mixins: [formatter],
	props: {
		notifications: { type: Array as PropType<Notification[]>, default: () => [] },
	},
	data() {
		return {
			vehicleStatus: reactive({}) as Record<string, VehicleStatus>,
		};
	},
	head() {
		return { title: this.$t("config.section.vehicles") };
	},
	computed: {
		uiLoadpoints(): UiLoadpoint[] {
			return store.uiLoadpoints.value;
		},
		stateVehicles(): Record<string, Vehicle> {
			return store.state?.vehicles || {};
		},
		cars(): CarState[] {
			const cars = new Map<string, CarState>();

			for (const [name, vehicle] of Object.entries(this.stateVehicles)) {
				const status = this.vehicleStatus[name];

				cars.set(name, {
					name,
					title: vehicle.title || name,
					icon: vehicle.icon || "car",
					loadpointTitle: "",
					lastUpdate: status?.updated || null,
					soc: status?.soc ?? null,
					capacity: vehicle.capacity ?? null,
					chargeStatus: status?.chargeStatus || null,
					vehicleLimitSoc: vehicle.limitSoc ?? null,
					range: status?.range ?? null,
					odometer: status?.odometer ?? null,
				});
			}

			for (const lp of this.uiLoadpoints) {
				if (!lp.vehicleName) {
					continue;
				}

				const current = cars.get(lp.vehicleName);
				const nextUpdate = lp.vehicleSocUpdated
					? new Date(lp.vehicleSocUpdated).getTime()
					: 0;
				const currentUpdate = current?.lastUpdate
					? new Date(current.lastUpdate).getTime()
					: 0;

				if (current && currentUpdate > nextUpdate) {
					continue;
				}

				const status = this.vehicleStatus[lp.vehicleName];
				const hasFreshVehicleData = Boolean(lp.vehicleSocUpdated);

				cars.set(lp.vehicleName, {
					name: lp.vehicleName,
					title: current?.title || lp.vehicleTitle || lp.vehicleName,
					icon: current?.icon || "car",
					loadpointTitle: lp.title || "",
					// prefer the loadpoint's poll time, fall back to when evcc last read the values
					lastUpdate: hasFreshVehicleData ? lp.vehicleSocUpdated : (status?.updated || null),
					soc: hasFreshVehicleData ? lp.vehicleSoc : (status?.soc ?? null),
					capacity: current?.capacity ?? null,
					// for connected cars the charger state is authoritative
					chargeStatus: lp.charging
						? "C"
						: lp.connected
							? "B"
							: status?.chargeStatus || "A",
					vehicleLimitSoc:
						hasFreshVehicleData && lp.vehicleLimitSoc > 0
							? lp.vehicleLimitSoc
							: (current?.vehicleLimitSoc ?? null),
					range:
						hasFreshVehicleData && lp.vehicleRange > 0
							? lp.vehicleRange
							: (status?.range ?? null),
					odometer:
						hasFreshVehicleData && lp.vehicleOdometer > 0
							? lp.vehicleOdometer
							: (status?.odometer ?? null),
				});
			}

			return [...cars.values()].sort((a, b) => a.title.localeCompare(b.title));
		},
	},
	mounted() {
		watch(
			() => Object.keys(this.stateVehicles).sort().join("\n"),
			() => this.loadVehicleStatus()
		);

		this.loadVehicleStatus();
	},
	methods: {
		async loadVehicleStatus() {
			if (store.state.offline) return;

			const names = Object.keys(this.stateVehicles);
			await Promise.all(
				names.map(async (name) => {
					try {
						const { data } = await api.get(`vehicles/${name}/status`);
						this.vehicleStatus[name] = data;
					} catch {
						// status is optional display info; ignore individual failures
					}
				})
			);
		},
		vehicleStatusTags(car: CarState): DeviceTagsMap {
			const values: Record<(typeof STATUS_KEYS)[number], number | string | null> = {
				soc: car.soc,
				capacity: car.capacity,
				chargeStatus: car.chargeStatus || "A",
				vehicleLimitSoc: car.vehicleLimitSoc,
				range: car.range,
				odometer: car.odometer,
			};

			return STATUS_KEYS.reduce((tags, key) => {
				const value = values[key];
				tags[key] = value === null ? { value: null, muted: true } : { value };
				return tags;
			}, {} as DeviceTagsMap);
		},
		formattedVehicleLastUpdate(car: CarState) {
			return car.lastUpdate ? this.fmtAbsoluteDate(new Date(car.lastUpdate)) : "";
		},
	},
});
</script>

<style scoped>
.car-loadpoint {
	padding-bottom: 0.25rem;
}

.car-meta {
	border-top: 1px solid var(--evcc-gray-25);
}

.label {
	min-width: 4rem;
}

.value {
	font-weight: bold;
	color: var(--bs-primary);
}

.value--muted {
	color: var(--evcc-gray) !important;
}
</style>
