<template>
	<div class="vehicle-soc">
		<div class="progress">
			<!-- Current SOC up to limit (green/primary) -->
			<div
				v-if="connected && socUpToLimitWidth > 0"
				class="progress-bar"
				role="progressbar"
				:class="{
					[progressColor]: true,
					'progress-bar-striped': charging && vehicleSoc <= visibleLimitSoc,
					'progress-bar-animated': charging && vehicleSoc <= visibleLimitSoc,
				}"
				:style="{ width: `${socUpToLimitWidth}%`, ...transition }"
			></div>
			<!-- Current SOC beyond limit (orange/warning) -->
			<div
				v-if="connected && socBeyondLimitWidth > 0"
				class="progress-bar bg-warning"
				role="progressbar"
				:class="{
					'progress-bar-striped': charging,
					'progress-bar-animated': charging,
				}"
				:style="{ width: `${socBeyondLimitWidth}%`, ...transition }"
			></div>
			<!-- Remaining to reach limit when below limit (grey/secondary) -->
			<div
				v-if="connected && remainingToLimitWidth > 0 && enabled"
				class="progress-bar bg-secondary"
				role="progressbar"
				:class="{
					'progress-bar-striped': charging,
					'progress-bar-animated': charging,
				}"
				:style="{ width: `${remainingToLimitWidth}%`, ...transition }"
			></div>
			<!-- Beyond limit to 100% (dark grey/muted) -->
			<div
				v-if="connected && beyondLimitWidth > 0"
				class="progress-bar bg-muted"
				role="progressbar"
				:style="{ width: `${beyondLimitWidth}%`, ...transition }"
			></div>
			<div
				v-show="vehicleLimitSoc"
				ref="vehicleLimitSoc"
				class="vehicle-limit-soc"
				data-bs-toggle="tooltip"
				title=" "
				:class="{ 'vehicle-limit-soc--active': vehicleLimitSocActive }"
				:style="{ left: `${vehicleLimitSoc}%` }"
			/>
			<div
				v-show="energyLimitMarkerPosition"
				class="energy-limit-marker"
				data-bs-toggle="tooltip"
				:class="{
					'energy-limit-marker--active': energyLimitMarkerActive,
					'energy-limit-marker--visible':
						energyLimitMarkerPosition !== null && energyLimitMarkerPosition < 100,
				}"
				:style="{ left: `${energyLimitMarkerPosition}%` }"
			/>
			<div
				v-show="planMarkerAvailable"
				class="plan-marker"
				data-bs-toggle="tooltip"
				:style="{ left: `${planMarkerPosition}%` }"
				data-testid="plan-marker"
				@click="$emit('plan-clicked')"
			>
				<shopicon-regular-clock></shopicon-regular-clock>
			</div>
		</div>
		<div class="target">
			<input
				v-if="socBasedCharging && connected"
				type="range"
				min="0"
				max="100"
				:step="step"
				:value="visibleLimitSoc"
				class="slider"
				:class="{ 'slider--active': sliderActive }"
				@mousedown="changeLimitSocStart"
				@touchstart="changeLimitSocStart"
				@input="movedLimitSoc"
				@mouseup="changeLimitSocEnd"
				@touchend="changeLimitSocEnd"
			/>
		</div>
	</div>
</template>

<script lang="ts">
import Tooltip from "bootstrap/js/dist/tooltip";
import "@h2d2/shopicons/es/regular/clock";
import formatter from "@/mixins/formatter";
import { defineComponent } from "vue";

export default defineComponent({
	name: "VehicleSoc",
	mixins: [formatter],
	props: {
		connected: Boolean,
		vehicleSoc: { type: Number, default: 0 },
		vehicleLimitSoc: { type: Number, default: 0 },
		enabled: Boolean,
		charging: Boolean,
		heating: Boolean,
		minSoc: { type: Number, default: 0 },
		effectivePlanSoc: { type: Number, default: 0 },
		effectiveLimitSoc: Number,
		limitEnergy: { type: Number, default: 0 },
		planEnergy: { type: Number, default: 0 },
		chargedEnergy: { type: Number, default: 0 },
		socBasedCharging: Boolean,
		socBasedPlanning: Boolean,
	},
	emits: ["limit-soc-drag", "limit-soc-updated", "plan-clicked"],
	data() {
		return {
			selectedLimitSoc: undefined as number | undefined,
			interactionStartScreenY: null,
			tooltip: null as Tooltip | null,
			dragging: false,
		};
	},
	computed: {
		step() {
			return this.heating ? 1 : 5;
		},
		vehicleSocDisplayWidth() {
			if (this.socBasedCharging) {
				if (this.vehicleSoc >= 0) {
					return this.vehicleSoc;
				}
				return 100;
			} else {
				if (this.maxEnergy) {
					return (100 / this.maxEnergy) * (this.chargedEnergy / 1e3);
				}
				return 100;
			}
		},
		socUpToLimitWidth() {
			// Green bar: current SOC up to limit, or just current SOC if no limit
			const currentSoc = this.vehicleSocDisplayWidth;
			const userLimit = this.visibleLimitSoc;

			if (!this.socBasedCharging) {
				return currentSoc;
			}

			// No limit set: show full current SOC in green
			if (!userLimit || userLimit === 0) {
				return currentSoc;
			}

			// Limit set: green bar goes up to minimum of (current SOC, limit)
			return Math.min(currentSoc, userLimit);
		},
		socBeyondLimitWidth() {
			// Orange bar: SOC beyond the user limit (only when SOC > limit)
			const currentSoc = this.vehicleSocDisplayWidth;
			const userLimit = this.visibleLimitSoc;

			if (!this.socBasedCharging || !userLimit || userLimit === 0) {
				return 0;
			}

			// Show orange only when current SOC exceeds the limit
			if (currentSoc > userLimit) {
				return currentSoc - userLimit;
			}

			return 0;
		},
		remainingToLimitWidth() {
			// Light grey bar: remaining to reach limit (only when SOC < limit)
			const currentSoc = this.vehicleSocDisplayWidth;
			const userLimit = this.visibleLimitSoc;

			if (!this.socBasedCharging || !userLimit || userLimit === 0) {
				return 0;
			}

			// Show light grey only when current SOC is below the limit
			if (currentSoc < userLimit) {
				return userLimit - currentSoc;
			}

			return 0;
		},
		beyondLimitWidth() {
			// Dark grey bar: remaining capacity from limit/SOC to 100%
			const currentSoc = this.vehicleSocDisplayWidth;
			const userLimit = this.visibleLimitSoc;

			if (!this.socBasedCharging) {
				return 0;
			}

			// No limit set: show grey from current SOC to 100%
			if (!userLimit || userLimit === 0) {
				return 100 - currentSoc;
			}

			// Limit set: show grey from max(SOC, limit) to 100%
			const startPoint = Math.max(currentSoc, userLimit);
			return 100 - startPoint;
		},
		transition() {
			if (this.dragging) {
				return { transition: "none" };
			}
			return { transition: "width var(--evcc-transition-fast) linear" };
		},
		maxEnergy() {
			return Math.max(this.planEnergy, this.limitEnergy, this.chargedEnergy / 1e3);
		},
		vehicleLimitSocActive() {
			return this.vehicleLimitSoc > 0 && this.vehicleLimitSoc > this.vehicleSoc;
		},
		planMarkerPosition(): number {
			if (this.socBasedPlanning) {
				return this.effectivePlanSoc;
			}
			const maxEnergy = Math.max(this.planEnergy, this.limitEnergy);
			if (maxEnergy) {
				return (100 / maxEnergy) * this.planEnergy;
			}
			return 0;
		},
		planMarkerAvailable() {
			if (this.socBasedCharging && !this.socBasedPlanning) {
				// mixed mode (% limit and kWh plan): hide marker
				return false;
			}
			return this.planMarkerPosition > 0;
		},
		energyLimitMarkerPosition() {
			if (this.socBasedCharging) {
				return null;
			}
			if (this.limitEnergy) {
				return (100 / this.maxEnergy) * this.limitEnergy;
			}
			return 100;
		},
		energyLimitMarkerActive() {
			if (this.socBasedCharging) {
				return false;
			}
			if (this.planEnergy) {
				return this.limitEnergy >= this.planEnergy;
			}
			return true;
		},
		sliderActive() {
			const isBelowVehicleLimit = this.visibleLimitSoc <= (this.vehicleLimitSoc || 100);
			const isAbovePlanLimit = this.visibleLimitSoc >= (this.effectivePlanSoc || 0);
			return isBelowVehicleLimit && isAbovePlanLimit;
		},
		progressColor() {
			if (this.minSocActive) {
				return "bg-danger";
			}
			return "bg-primary";
		},
		minSocActive() {
			return this.minSoc > 0 && this.vehicleSoc < this.minSoc;
		},
		visibleLimitSoc() {
			const value = Number(this.selectedLimitSoc || this.effectiveLimitSoc);
			return isNaN(value) ? 0 : value;
		},
	},
	watch: {
		effectiveLimitSoc() {
			this.selectedLimitSoc = this.effectiveLimitSoc;
		},
		vehicleLimitSoc() {
			this.updateTooltip();
		},
	},
	mounted() {
		this.updateTooltip();
	},
	methods: {
		changeLimitSocStart(e: Event) {
			this.dragging = true;
			e.stopPropagation();
		},
		changeLimitSocEnd(e: Event) {
			this.dragging = false;
			const value = parseInt((e.target as HTMLInputElement).value, 10);
			// value changed
			if (value !== this.effectiveLimitSoc) {
				this.$emit("limit-soc-updated", value);
			}
		},
		movedLimitSoc(e: Event) {
			const value = parseInt((e.target as HTMLInputElement).value, 10);
			e.stopPropagation();
			const minLimit = 20;
			if (value < minLimit) {
				(e.target as HTMLInputElement).value = minLimit.toString();
				this.selectedLimitSoc = value;
				e.preventDefault();
				return false;
			}
			this.selectedLimitSoc = value;

			this.$emit("limit-soc-drag", this.selectedLimitSoc);
			return true;
		},
		updateTooltip() {
			if (!this.tooltip) {
				this.tooltip = new Tooltip(this.$refs["vehicleLimitSoc"] as HTMLElement);
			}
			const value = this.heating
				? this.fmtTemperature(this.vehicleLimitSoc)
				: this.fmtPercentage(this.vehicleLimitSoc);
			const key = this.heating ? "heatingStatus" : "vehicleStatus";
			const content = `${this.$t(`main.${key}.vehicleLimit`)}: ${value}`;
			this.tooltip.setContent({ ".tooltip-inner": content });
		},
	},
});
</script>
<style scoped>
.vehicle-soc {
	--height: 32px;
	--thumb-overlap: 6px;
	--thumb-width: 12px;
	--label-height: 26px;
	position: relative;
	height: var(--height);
}
.progress {
	height: 100%;
	font-size: 1rem;
	background: var(--evcc-background);
}
.progress-bar {
	background-color: var(--bs-primary) !important;
}
.progress-bar.bg-primary {
	background-color: var(--bs-primary) !important;
}
.progress-bar.bg-secondary {
	background-color: var(--bs-gray-medium) !important;
	opacity: 0.5;
}
.progress-bar.bg-warning {
	background-color: var(--evcc-orange) !important;
}
.progress-bar.bg-danger {
	background-color: var(--bs-danger) !important;
}
.progress-bar.bg-muted {
	background-color: var(--evcc-background) !important;
	opacity: 1;
}
.bg-light {
	color: var(--bs-gray-dark);
}
.slider {
	-webkit-appearance: none;
	position: absolute;
	top: calc(var(--thumb-overlap) * -1);
	height: calc(100% + 2 * var(--thumb-overlap));
	width: 100%;
	background: transparent;
	pointer-events: none;
}
.slider:focus {
	outline: none;
}
/* Note: Safari,Chrome,Blink and Firefox specific styles need to be in separate definitions to work */
.slider::-webkit-slider-runnable-track {
	position: relative;
	background: transparent;
	border: none;
	height: 100%;
	cursor: auto;
	/* ensure slider start/end is at the edge and not inside the track */
	margin: 0 -6px;
}
.slider::-moz-range-track {
	background: transparent;
	border: none;
	height: 100%;
	cursor: auto;
	/* ensure slider start/end is at the edge and not inside the track */
	margin: 0 -6px;
}
.slider::-webkit-slider-thumb {
	-webkit-appearance: none;
	position: relative;
	margin-left: var(--thumb-width) / 2;
	height: 100%;
	width: var(--thumb-width);
	background-color: var(--evcc-gray);
	cursor: grab;
	border: none;
	opacity: 1;
	border-radius: var(--thumb-overlap);
	box-shadow: 0 0 6px var(--evcc-background);
	pointer-events: auto;
	transition: background-color var(--evcc-transition-fast) linear;
}
.slider::-moz-range-thumb {
	position: relative;
	height: 100%;
	width: var(--thumb-width);
	background-color: var(--evcc-gray);
	cursor: grab;
	border: none;
	opacity: 1;
	border-radius: var(--thumb-overlap);
	box-shadow: 0 0 6px var(--evcc-background);
	pointer-events: auto;
	transition: background-color var(--evcc-transition-fast) linear;
}
.slider--active::-webkit-slider-thumb {
	background-color: var(--evcc-dark-green);
}
.slider--active::-moz-range-thumb {
	background-color: var(--evcc-dark-green);
}
.vehicle-limit-soc {
	position: absolute;
	top: 0;
	bottom: 0;
	width: 20px;
	transform: translateX(-8px);
	background-color: transparent;
	background-clip: padding-box;
	border-width: 0 8px;
	border-style: solid;
	border-color: transparent;
	transition-property: background-color, left;
	transition-timing-function: linear;
	transition-duration: var(--evcc-transition-fast);
}
.vehicle-limit-soc--active {
	background-color: var(--evcc-box);
}
.plan-marker {
	position: absolute;
	top: 0;
	transform: translateX(-50%);
	color: var(--evcc-darker-green);
	transition-property: color, left, opacity;
	transition-timing-function: linear;
	cursor: pointer;
	transition-duration: var(--evcc-transition-fast);
}
.plan-marker::before {
	content: "";
	display: block;
	height: var(--height);
	border-width: 0 10px;
	background-clip: padding-box;
	border-style: solid;
	border-color: transparent;
	background-color: var(--evcc-darker-green);
	transition: background-color var(--evcc-transition-fast) linear;
}
.energy-limit-marker {
	position: absolute;
	top: -4px;
	bottom: -4px;
	transform: translateX(-50%);
	width: 4px;
	opacity: 0;
	background-color: var(--evcc-gray);
	transition-property: opacity, left, background-color;
	transition-timing-function: linear;
	transition-duration: var(--evcc-transition-fast);
}
.energy-limit-marker--active {
	background-color: var(--evcc-dark-green);
}
.energy-limit-marker--visible {
	opacity: 1;
}
</style>
