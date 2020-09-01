# Prometheus Remote Write Exporter

# OpenTelemetry Go SDK Prometheus Remote Write Exporter

## Table of Contents

- [Architecture Overview](#architecture-overview)
  - [Data Path](#data-path)
- [Oustanding Tasks](#oustanding-tasks)
- [Pull Requests Filed and Merged](#pull-requests-filed-and-merged)
- [Reference Documents](#reference-documents)
- [Contributors](#contributors)

## Architecture Overview
The design docuement and README can be found in upstream repository. 
![Exporter UML Diagram](./img/exporter-uml.png)

### Data Path

`Processor`. The CheckpointSet is then sent to the `Export` when Export() is called.

#### Exporter Data Path

![exporter-data-pipeline Pipeline](./img/exporter-sequence.png)

The Exporter receives a pdata.Metrics from the pipeline, converts the metrics to
TimeSeries, and sends them in a snappy-compressed message via HTTP to Cortex.

## Repository Structure
See this [link](https://github.com/open-telemetry/opentelemetry-collector/tree/master/exporter/prometheusremotewriteexporter)

## Outstanding Tasks

We have filed several issues for enhancements to the Exporter:

- List issues
- here

## Pull Requests Filed and Merged

## Reference Documents

## Contributors

- [Yang Hu](https://github.com/huyan0)
