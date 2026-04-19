# Graph Report - .  (2026-04-19)

## Corpus Check
- 161 files · ~83,152 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 910 nodes · 2236 edges · 90 communities detected
- Extraction: 53% EXTRACTED · 47% INFERRED · 0% AMBIGUOUS · INFERRED: 1040 edges (avg confidence: 0.74)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]
- [[_COMMUNITY_Community 7|Community 7]]
- [[_COMMUNITY_Community 8|Community 8]]
- [[_COMMUNITY_Community 9|Community 9]]
- [[_COMMUNITY_Community 10|Community 10]]
- [[_COMMUNITY_Community 11|Community 11]]
- [[_COMMUNITY_Community 12|Community 12]]
- [[_COMMUNITY_Community 13|Community 13]]
- [[_COMMUNITY_Community 14|Community 14]]
- [[_COMMUNITY_Community 15|Community 15]]
- [[_COMMUNITY_Community 16|Community 16]]
- [[_COMMUNITY_Community 17|Community 17]]
- [[_COMMUNITY_Community 18|Community 18]]
- [[_COMMUNITY_Community 19|Community 19]]
- [[_COMMUNITY_Community 20|Community 20]]
- [[_COMMUNITY_Community 21|Community 21]]
- [[_COMMUNITY_Community 22|Community 22]]
- [[_COMMUNITY_Community 23|Community 23]]
- [[_COMMUNITY_Community 24|Community 24]]
- [[_COMMUNITY_Community 25|Community 25]]
- [[_COMMUNITY_Community 26|Community 26]]
- [[_COMMUNITY_Community 27|Community 27]]
- [[_COMMUNITY_Community 28|Community 28]]
- [[_COMMUNITY_Community 29|Community 29]]
- [[_COMMUNITY_Community 30|Community 30]]
- [[_COMMUNITY_Community 31|Community 31]]
- [[_COMMUNITY_Community 32|Community 32]]
- [[_COMMUNITY_Community 33|Community 33]]
- [[_COMMUNITY_Community 34|Community 34]]
- [[_COMMUNITY_Community 35|Community 35]]
- [[_COMMUNITY_Community 36|Community 36]]
- [[_COMMUNITY_Community 37|Community 37]]
- [[_COMMUNITY_Community 38|Community 38]]
- [[_COMMUNITY_Community 39|Community 39]]
- [[_COMMUNITY_Community 40|Community 40]]
- [[_COMMUNITY_Community 41|Community 41]]
- [[_COMMUNITY_Community 42|Community 42]]
- [[_COMMUNITY_Community 43|Community 43]]
- [[_COMMUNITY_Community 44|Community 44]]
- [[_COMMUNITY_Community 45|Community 45]]
- [[_COMMUNITY_Community 46|Community 46]]
- [[_COMMUNITY_Community 47|Community 47]]
- [[_COMMUNITY_Community 48|Community 48]]
- [[_COMMUNITY_Community 49|Community 49]]
- [[_COMMUNITY_Community 50|Community 50]]
- [[_COMMUNITY_Community 51|Community 51]]
- [[_COMMUNITY_Community 52|Community 52]]
- [[_COMMUNITY_Community 53|Community 53]]
- [[_COMMUNITY_Community 54|Community 54]]
- [[_COMMUNITY_Community 55|Community 55]]
- [[_COMMUNITY_Community 56|Community 56]]
- [[_COMMUNITY_Community 57|Community 57]]
- [[_COMMUNITY_Community 58|Community 58]]
- [[_COMMUNITY_Community 59|Community 59]]
- [[_COMMUNITY_Community 60|Community 60]]
- [[_COMMUNITY_Community 61|Community 61]]
- [[_COMMUNITY_Community 62|Community 62]]
- [[_COMMUNITY_Community 63|Community 63]]
- [[_COMMUNITY_Community 64|Community 64]]
- [[_COMMUNITY_Community 65|Community 65]]
- [[_COMMUNITY_Community 66|Community 66]]
- [[_COMMUNITY_Community 67|Community 67]]
- [[_COMMUNITY_Community 68|Community 68]]
- [[_COMMUNITY_Community 69|Community 69]]
- [[_COMMUNITY_Community 70|Community 70]]
- [[_COMMUNITY_Community 71|Community 71]]
- [[_COMMUNITY_Community 72|Community 72]]
- [[_COMMUNITY_Community 73|Community 73]]
- [[_COMMUNITY_Community 74|Community 74]]
- [[_COMMUNITY_Community 75|Community 75]]
- [[_COMMUNITY_Community 76|Community 76]]
- [[_COMMUNITY_Community 77|Community 77]]
- [[_COMMUNITY_Community 78|Community 78]]
- [[_COMMUNITY_Community 79|Community 79]]
- [[_COMMUNITY_Community 80|Community 80]]
- [[_COMMUNITY_Community 81|Community 81]]
- [[_COMMUNITY_Community 82|Community 82]]
- [[_COMMUNITY_Community 83|Community 83]]
- [[_COMMUNITY_Community 84|Community 84]]
- [[_COMMUNITY_Community 85|Community 85]]
- [[_COMMUNITY_Community 86|Community 86]]
- [[_COMMUNITY_Community 87|Community 87]]
- [[_COMMUNITY_Community 88|Community 88]]
- [[_COMMUNITY_Community 89|Community 89]]

## God Nodes (most connected - your core abstractions)
1. `CatalogIngestError` - 53 edges
2. `Harness` - 45 edges
3. `LoadedClipInput` - 31 edges
4. `CatalogRepository` - 26 edges
5. `TestE2E_RecommendationSelectorRespectsFallbackMaxAndCoreDominantMin()` - 23 edges
6. `TestE2E_RecommendationSelectorRespectsFutureLikeMaxInLowSupply()` - 23 edges
7. `IngestionRecordPayload` - 23 edges
8. `testDB()` - 22 edges
9. `TestE2E_RecommendationWritesAuditAndServingStateWithEvidence()` - 22 edges
10. `TestE2E_RecommendationDemandMapping_SuspendedInactiveAndNonTargetUnitsAreExcluded()` - 22 edges

## Surprising Connections (you probably didn't know these)
- `Module Registry` --conceptually_related_to--> `Three-Domain Boundary`  [INFERRED]
  cmd/dbtool/specs.go → docs/全新设计-总设计.md
- `Registry Contract Tests` --references--> `Recommendation Materialized Views`  [EXTRACTED]
  cmd/dbtool/modules_test.go → docs/全新设计-推荐模块设计.md
- `Module Migration Engine` --conceptually_related_to--> `Module Boundary Rules`  [INFERRED]
  cmd/dbtool/migrator.go → AGENTS.md
- `Module Registry` --references--> `Recommendation Materialized Views`  [EXTRACTED]
  cmd/dbtool/specs.go → docs/全新设计-推荐模块设计.md
- `dbtool CLI` --references--> `Recommendation Materialized Views`  [EXTRACTED]
  cmd/dbtool/main.go → docs/全新设计-推荐模块设计.md

## Hyperedges (group relationships)
- **Authoritative Design Set** — zongsheji_overall_design_doc, learningdesign_learning_engine_doc, recommenddesign_recommendation_doc, catalogdesign_catalog_doc [EXTRACTED 1.00]
- **Three-Domain Architecture** — learningdesign_learning_engine_doc, catalogdesign_catalog_doc, recommenddesign_recommendation_doc [EXTRACTED 1.00]
- **dbtool Module Migration Workflow** — main_dbtool_cli, migrator_module_migration_engine, specs_module_registry [EXTRACTED 1.00]
- **Learningengine Record Pipeline** — le_service_record_learning_events_usecase, le_service_tx_manager, le_repo_unit_learning_event_repository, le_repo_user_unit_state_repository, le_service_learning_state_reducer [EXTRACTED 1.00]
- **Learningengine Replay Pipeline** — le_service_replay_user_states_usecase, le_service_tx_manager, le_repo_user_unit_state_repository, le_repo_unit_learning_event_repository, le_service_control_snapshot_merge, le_service_learning_state_reducer [EXTRACTED 1.00]
- **Recommendation Main Pipeline** — rec_steps_generate_video_recommendations, rec_steps_context_assembler, rec_steps_demand_planner, rec_steps_candidate_generator, rec_steps_evidence_resolver, rec_steps_video_evidence_aggregator, rec_steps_video_ranker, rec_steps_video_selector, rec_steps_explanation_builder, rec_steps_serving_state_manager, rec_steps_audit_writer [EXTRACTED 1.00]
- **Learning State Reduction Pipeline** — reducer_user_unit_state_reducer, event_learning_event_validation, progression_sm2_progression_policy, progression_active_status_policy, progression_progress_and_mastery_metrics [EXTRACTED 1.00]
- **State Persistence Mapping Contract** — learning_event_learning_event_model, user_unit_state_user_unit_state_model, models_learning_event_row_mapper, models_user_unit_state_row_mapper, pgtypes_postgres_type_converters [EXTRACTED 1.00]
- **Target Unit Command Suite** — target_unit_commands_ensure_target_units_usecase, target_unit_commands_set_target_inactive_usecase, target_unit_commands_suspend_target_unit_usecase, target_unit_commands_resume_target_unit_usecase [EXTRACTED 1.00]
- **Recommendation Read Ports** — learning_state_reader_port, recommendable_video_unit_reader_port, unit_inventory_reader_port, video_user_state_reader_port, evidence_readers_semantic_span_reader, evidence_readers_transcript_sentence_reader [EXTRACTED 1.00]
- **Recommendation Final Write Bundle** — recommendation_audit_repository_port, serving_state_unit_repository_port, serving_state_video_repository_port [EXTRACTED 1.00]
- **Learningengine Transaction Repository Bundle** — tx_manager_manager, target_state_command_repository_impl, unit_learning_event_repository_impl, user_unit_state_repository_impl [EXTRACTED 1.00]
- **Candidate Generation Lane Mix** — default_candidate_generator_default_candidate_generator, default_candidate_generator_exact_core_lane, default_candidate_generator_bundle_lane, default_candidate_generator_soft_future_lane, default_candidate_generator_quality_fallback_lane [EXTRACTED 1.00]
- **Recommendation Execution Pipeline** — generate_video_recommendations_impl_generate_video_recommendations_service, context_assembler_context_assembler_interface, candidate_generator_candidate_generator_interface, video_evidence_aggregator_video_evidence_aggregator_interface, explanation_builder_explanation_builder_interface, side_effects_recommendation_result_writer_interface [EXTRACTED 1.00]
- **Transactional Recommendation Persistence** — default_result_writer_default_recommendation_result_writer, side_effects_audit_writer_interface, side_effects_serving_state_manager_interface, tx_context_sqlc_queries_context [EXTRACTED 1.00]
- **Recommendation Context State Assembly** — recommendation_context_recommendation_context, request_recommendation_request, learning_state_snapshot_learning_state_snapshot, unit_video_inventory_unit_video_inventory, serving_state_user_unit_serving_state, serving_state_user_video_serving_state, video_user_state_video_user_state, recommendable_video_unit_recommendable_video_unit [EXTRACTED 1.00]
- **Demand Bundle Plan Structure** — demand_models_demand_bundle, demand_models_demand_unit, demand_models_lane_budget, demand_models_mix_quota, demand_models_planner_flags [EXTRACTED 1.00]
- **Ranking Penalty Pipeline** — default_video_ranker_default_video_ranker, default_video_ranker_freshness_score, default_video_ranker_recent_served_penalty, default_video_ranker_recent_watched_penalty, default_video_ranker_overload_penalty [EXTRACTED 1.00]
- **Selector Marginal-Coverage Pipeline** — default_video_selector_default_video_selector, default_video_selector_selector_mode_gate, default_video_selector_core_dominant_bootstrap, default_video_selector_marginal_coverage_selection, default_video_selector_mix_quota_constraints [EXTRACTED 1.00]
- **Recommendation Repository Read-Side Bundle** — recommendation_repository_learning_state_reader, recommendation_repository_recommendable_video_unit_reader, recommendation_repository_unit_inventory_reader, recommendation_repository_semantic_span_reader, recommendation_repository_transcript_sentence_reader, recommendation_repository_video_user_state_reader [EXTRACTED 1.00]
- **Recommendation Repository Write-Side Bundle** — recommendation_repository_audit_repository, recommendation_repository_serving_state_repositories, recommendation_sqlc_audit_insert_queries, recommendation_sqlc_serving_state_upsert_queries [EXTRACTED 1.00]
- **Recommendation SQLC Read-Model Contract** — recommendation_sqlc_queries_wrapper, recommendation_sqlc_projection_models, recommendation_sqlc_querier_contract, recommendation_sqlc_read_models_queries, recommendation_sqlc_write_queries [EXTRACTED 1.00]
- **Recommendation Projection Mapping Contract** — recommendation_mapper_learning_state_snapshot_mapper, recommendation_mapper_recommendable_video_unit_mapper, recommendation_mapper_unit_video_inventory_mapper, recommendation_mapper_evidence_projection_mappers, recommendation_mapper_serving_state_mappers, recommendation_mapper_pgtype_converters [EXTRACTED 1.00]
- **Recommendation Unit Test Matrix** — recommendation_candidate_generator_test_suite, recommendation_context_assembler_test_suite, recommendation_evidence_resolver_test_suite, recommendation_pipeline_usecase_test_suite, recommendation_shell_usecase_test_suite, recommendation_aggregator_test_suite, recommendation_explanation_builder_test_suite, recommendation_demand_planner_test_suite, recommendation_video_ranker_test_suite, recommendation_video_selector_test_suite [EXTRACTED 1.00]
- **Recommendation Integration Test Bundle** — recommendation_integration_fixture_embedded_postgres, recommendation_integration_fixture_step1_schema, recommendation_repository_integration_suite, recommendation_tx_manager [EXTRACTED 1.00]
- **Cross-Module Recommendation E2E Bundle** — cross_module_e2e_test_scope, cross_module_e2e_harness, cross_module_e2e_learning_to_recommendation_suite, cross_module_e2e_recommendation_audit_serving_suite, cross_module_e2e_recommendation_supply_modes_suite, cross_module_e2e_suite_bootstrap, cross_module_e2e_helper_builders [EXTRACTED 1.00]

## Communities

### Community 0 - "Community 0"
Cohesion: 0.05
Nodes (116): Exception, build_normalized_clip_data(), _build_transcript_row(), _build_unit_index_rows(), _dedupe_surface_forms(), _evidence_pick_key(), _merge_intervals_and_measure(), 按当前设计规则选出稳定的 evidence spans。 (+108 more)

### Community 1 - "Community 1"
Cohesion: 0.04
Nodes (74): TestDatabase, execer, TestManagerRollsBackTransactionOnError(), TestManagerWithinUserTxAllowsDifferentUsersConcurrently(), TestManagerWithinUserTxSerializesSameUser(), NewManager(), flattenGroupedEvents(), groupAndSortEvents() (+66 more)

### Community 2 - "Community 2"
Cohesion: 0.13
Nodes (53): failingAuditWriter, failNow(), MustEnsureTarget(), MustRecommend(), assertAnyVideoCoversUnit(), assertContainsVideo(), assertContiguousRanks(), assertCoveredUnits() (+45 more)

### Community 3 - "Community 3"
Cohesion: 0.05
Nodes (37): buildAuditPayload(), candidateSummary(), hasDemand(), lanePriority(), mapFinalItems(), NewGenerateVideoRecommendationsPipeline(), newRunID(), primaryLane() (+29 more)

### Community 4 - "Community 4"
Cohesion: 0.12
Nodes (42): appendUnique(), containsVideo(), filterCandidatesByLane(), orderedDistinctVideos(), recommendableRow(), recommendationContext(), recommendationDemand(), summarizeCandidates() (+34 more)

### Community 5 - "Community 5"
Cohesion: 0.12
Nodes (38): Agent Rules, Module Boundary Rules, Catalog Delta Migration Doc, Deprecated Catalog Design Doc, Deprecated Learning Engine Doc, Deprecated Overall MVP Doc, Historical Docs Index, Deprecated Recommendation Scheduler Doc (+30 more)

### Community 6 - "Community 6"
Cohesion: 0.09
Nodes (21): NewDefaultContextAssembler(), normalizeRequest(), TestDefaultContextAssemblerAssembleAppliesDefaultsAndLoadsDependencies(), TestDefaultContextAssemblerAssembleReturnsErrors(), TestNormalizeDurationResetsInvalidRange(), uniqueUnitIDs(), NewDefaultDemandPlanner(), NewDefaultVideoStateEnricher() (+13 more)

### Community 7 - "Community 7"
Cohesion: 0.08
Nodes (32): Evidence Projection Mappers, Learning State Snapshot Mapper, Recommendation Postgres Type Converters, Recommendable Video Unit Mapper, Serving State Mappers, Unit Video Inventory Mapper, Recommendation Materialized Read Models, Recommendation Owner Boundary (+24 more)

### Community 8 - "Community 8"
Cohesion: 0.08
Nodes (32): Evidence Ref, Resolved Evidence Window, Video Candidate, Video Unit Candidate, Demand Bundle, Demand Unit, Lane Budget, Mix Quota (+24 more)

### Community 9 - "Community 9"
Cohesion: 0.09
Nodes (27): Catalog Clean Baseline Migration Policy, Catalog Module, Finding F-REC-001, Finding F-REC-002, Finding F-REC-003, Finding F-REC-004, docs/全新设计-推荐模块设计.md, docs/全新设计-总设计.md (+19 more)

### Community 10 - "Community 10"
Cohesion: 0.18
Nodes (22): canSelect(), countCoreDominant(), countFallback(), countFutureDominant(), countFutureLike(), countUncovered(), incrementDominantUnitCount(), isCoreDominant() (+14 more)

### Community 11 - "Community 11"
Cohesion: 0.21
Nodes (18): appendUniqueInt32(), NewDefaultEvidenceResolver(), parseEvidenceRefs(), resolveBestBounds(), resolveWindowBounds(), resolveWindowSentenceIndexes(), selectBestEvidence(), int64Ptr() (+10 more)

### Community 12 - "Community 12"
Cohesion: 0.12
Nodes (19): applySchemaSequence(), freePort(), migrationFiles(), migrationFilesForMain(), nullIfEmpty(), OpenHarness(), repoRoot(), repoRootFromRuntime() (+11 more)

### Community 13 - "Community 13"
Cohesion: 0.25
Nodes (17): bucketPriority(), freshnessScore(), NewDefaultVideoRanker(), overloadPenalty(), recencyPenalty(), recentServedPenalty(), recentWatchedPenalty(), round4() (+9 more)

### Community 14 - "Community 14"
Cohesion: 0.25
Nodes (12): execer, Suite, applyLearningEngineSchema(), applyRecommendationSchema(), execSQLFile(), freePort(), migrationFiles(), migrationVersion() (+4 more)

### Community 15 - "Community 15"
Cohesion: 0.25
Nodes (17): finalizeState(), initState(), int16Pointer(), Reduce(), emptyState(), learningEvent(), masteredState(), TestReduce_FailureAfterMasteredFallsBackToReviewing() (+9 more)

### Community 16 - "Community 16"
Cohesion: 0.23
Nodes (14): bucketBaseWeight(), ceilFraction(), classifyDemandUnit(), floorFraction(), isHardReview(), isSoftReview(), plannerFlags(), plannerLaneBudget() (+6 more)

### Community 17 - "Community 17"
Cohesion: 0.17
Nodes (16): Stable Learning Engine Enums, Weak vs Strong Event Classification, Learning Event Validation, Learning Event Types, Learning Event, Learning Event Row Mapper, User Unit State Row Mapper, Postgres Type Converters (+8 more)

### Community 18 - "Community 18"
Cohesion: 0.33
Nodes (7): LearningStateReader, learning.unit_learning_events, learning.user_unit_states, sqlc Query Facade, TargetStateCommandRepository, UnitLearningEventRepository, UserUnitStateRepository

### Community 19 - "Community 19"
Cohesion: 0.4
Nodes (6): Default Audit Writer, Transaction-Aware Audit Persistence, Atomic Recommendation Persistence, Default Recommendation Result Writer, Default Serving State Manager, SQLC Queries Context

### Community 20 - "Community 20"
Cohesion: 0.33
Nodes (5): AuthUser, CatalogVideo, LearningUnitLearningEvent, LearningUserUnitState, SemanticCoarseUnit

### Community 21 - "Community 21"
Cohesion: 0.4
Nodes (4): BestEvidence, GenerateVideoRecommendationsRequest, GenerateVideoRecommendationsResponse, RecommendationVideo

### Community 22 - "Community 22"
Cohesion: 0.4
Nodes (4): AuditWriter, RecommendationResultWriter, ServingStateManager, VideoStateEnricher

### Community 23 - "Community 23"
Cohesion: 0.5
Nodes (4): Current Final Baseline Only, Learning Engine Migrations, Learning Engine Owner Boundary, learningengine_schema_migrations Tracking Table

### Community 24 - "Community 24"
Cohesion: 0.5
Nodes (4): catalog.video_user_states, Recommendation Boundary, Recommendation Module, Video Recommendation Pipeline

### Community 25 - "Community 25"
Cohesion: 0.67
Nodes (3): Default Explanation Builder, Reason-Code Explanation Generation, Explanation Builder Interface

### Community 26 - "Community 26"
Cohesion: 0.67
Nodes (3): Default Video Evidence Aggregator, Video-Level Evidence Aggregation, Video Evidence Aggregator Interface

### Community 27 - "Community 27"
Cohesion: 0.67
Nodes (3): ListUserUnitStatesRequest, UserUnitStateRepository Port, ListUserUnitStatesUsecase

### Community 28 - "Community 28"
Cohesion: 0.67
Nodes (2): TransactionalRepositories, TxManager

### Community 29 - "Community 29"
Cohesion: 0.67
Nodes (3): Finding F-LE-001, docs/全新设计-学习引擎设计.md, User-level Replay/Write Mutex

### Community 30 - "Community 30"
Cohesion: 1.0
Nodes (2): RecommendableVideoUnitReader, UnitInventoryReader

### Community 31 - "Community 31"
Cohesion: 1.0
Nodes (2): SemanticSpanReader, TranscriptSentenceReader

### Community 32 - "Community 32"
Cohesion: 1.0
Nodes (2): Recommendation Transaction Manager, WithinTx Flow

### Community 33 - "Community 33"
Cohesion: 1.0
Nodes (1): CandidateGenerator

### Community 34 - "Community 34"
Cohesion: 1.0
Nodes (2): Recommendation Selection Logic, Video Selector Interface

### Community 35 - "Community 35"
Cohesion: 1.0
Nodes (1): EvidenceResolver

### Community 36 - "Community 36"
Cohesion: 1.0
Nodes (1): Querier

### Community 37 - "Community 37"
Cohesion: 1.0
Nodes (2): Embedded Postgres Test Deviation, Real Postgres Test Layout

### Community 38 - "Community 38"
Cohesion: 1.0
Nodes (1): DATABASE_URL Loader

### Community 39 - "Community 39"
Cohesion: 1.0
Nodes (0): 

### Community 40 - "Community 40"
Cohesion: 1.0
Nodes (1): Explanation Builder Unit Test Suite

### Community 41 - "Community 41"
Cohesion: 1.0
Nodes (1): Demand Planner Unit Test Suite

### Community 42 - "Community 42"
Cohesion: 1.0
Nodes (1): Video Evidence Aggregator Unit Test Suite

### Community 43 - "Community 43"
Cohesion: 1.0
Nodes (1): VideoUserStateReader

### Community 44 - "Community 44"
Cohesion: 1.0
Nodes (1): RecommendationAuditRepository

### Community 45 - "Community 45"
Cohesion: 1.0
Nodes (1): UnitServingStateRepository

### Community 46 - "Community 46"
Cohesion: 1.0
Nodes (1): VideoServingStateRepository

### Community 47 - "Community 47"
Cohesion: 1.0
Nodes (1): Generate Video Recommendations Usecase

### Community 48 - "Community 48"
Cohesion: 1.0
Nodes (0): 

### Community 49 - "Community 49"
Cohesion: 1.0
Nodes (0): 

### Community 50 - "Community 50"
Cohesion: 1.0
Nodes (0): 

### Community 51 - "Community 51"
Cohesion: 1.0
Nodes (1): Context Assembler Interface

### Community 52 - "Community 52"
Cohesion: 1.0
Nodes (0): 

### Community 53 - "Community 53"
Cohesion: 1.0
Nodes (0): 

### Community 54 - "Community 54"
Cohesion: 1.0
Nodes (1): Evidence Resolution Scope

### Community 55 - "Community 55"
Cohesion: 1.0
Nodes (1): Learning-to-Recommendation Replay Consistency E2E

### Community 56 - "Community 56"
Cohesion: 1.0
Nodes (1): Shared E2E Harness Bootstrap

### Community 57 - "Community 57"
Cohesion: 1.0
Nodes (1): Content Facts and Recall-ready Indexes

### Community 58 - "Community 58"
Cohesion: 1.0
Nodes (1): TargetUnitSpec

### Community 59 - "Community 59"
Cohesion: 1.0
Nodes (1): RecordLearningEventsRequest

### Community 60 - "Community 60"
Cohesion: 1.0
Nodes (1): ReplayUserStatesRequest

### Community 61 - "Community 61"
Cohesion: 1.0
Nodes (1): TargetStateCommandRepository Port

### Community 62 - "Community 62"
Cohesion: 1.0
Nodes (1): UnitLearningEventRepository Port

### Community 63 - "Community 63"
Cohesion: 1.0
Nodes (1): ErrLateStrongEvent

### Community 64 - "Community 64"
Cohesion: 1.0
Nodes (1): List User Unit States Usecase

### Community 65 - "Community 65"
Cohesion: 1.0
Nodes (1): Ensure Target Units Usecase

### Community 66 - "Community 66"
Cohesion: 1.0
Nodes (1): Set Target Inactive Usecase

### Community 67 - "Community 67"
Cohesion: 1.0
Nodes (1): Resume Target Unit Usecase

### Community 68 - "Community 68"
Cohesion: 1.0
Nodes (1): Record Learning Events Usecase

### Community 69 - "Community 69"
Cohesion: 1.0
Nodes (1): Replay User States Usecase

### Community 70 - "Community 70"
Cohesion: 1.0
Nodes (1): Reducer and Aggregate Root Package Scope

### Community 71 - "Community 71"
Cohesion: 1.0
Nodes (1): User Unit State Filter

### Community 72 - "Community 72"
Cohesion: 1.0
Nodes (1): Target Unit Spec

### Community 73 - "Community 73"
Cohesion: 1.0
Nodes (1): 生成统一审计上下文。          审计上下文尽量只放排障需要的信息，不放整坨 transcript。

### Community 74 - "Community 74"
Cohesion: 1.0
Nodes (0): 

### Community 75 - "Community 75"
Cohesion: 1.0
Nodes (1): 表示 normalizer 阶段产出的基础行集合。      这里故意不包含 transcript 摘要和 unit index。     原因是这两类数据属于

### Community 76 - "Community 76"
Cohesion: 1.0
Nodes (1): 表示 normalizer 和 index_builder 产出的完整写库数据。

### Community 77 - "Community 77"
Cohesion: 1.0
Nodes (1): 表示写入审计表时需要的字段集合。      这里不包含 ingestion_record_id 和时间戳，因为这两个值应由 repository 在真正写库时生

### Community 78 - "Community 78"
Cohesion: 1.0
Nodes (1): 表示数据库里已存在的 clip 快照。      这个对象专门给 main 做“是否可以 skipped”判断用。     它只保留幂等判断所需的字段，不承担完

### Community 79 - "Community 79"
Cohesion: 1.0
Nodes (1): 表示 main 汇总时使用的单 clip 最终结果。

### Community 80 - "Community 80"
Cohesion: 1.0
Nodes (1): Single Reducer Rule

### Community 81 - "Community 81"
Cohesion: 1.0
Nodes (1): RecordLearningEvents Pipeline

### Community 82 - "Community 82"
Cohesion: 1.0
Nodes (1): ReplayUserStates Pipeline

### Community 83 - "Community 83"
Cohesion: 1.0
Nodes (1): Short Final-Write Transactions

### Community 84 - "Community 84"
Cohesion: 1.0
Nodes (1): Run/Item Audit Center Policy

### Community 85 - "Community 85"
Cohesion: 1.0
Nodes (1): Read-Only Upstream Data Policy

### Community 86 - "Community 86"
Cohesion: 1.0
Nodes (1): video_recommendation_runs / video_recommendation_items

### Community 87 - "Community 87"
Cohesion: 1.0
Nodes (1): user_unit_serving_states

### Community 88 - "Community 88"
Cohesion: 1.0
Nodes (1): user_video_serving_states

### Community 89 - "Community 89"
Cohesion: 1.0
Nodes (1): Cross-Module E2E Test Scope

## Knowledge Gaps
- **141 isolated node(s):** `DATABASE_URL Loader`, `candidateSummary`, `Explanation Builder Unit Test Suite`, `Demand Planner Unit Test Suite`, `Video Evidence Aggregator Unit Test Suite` (+136 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **Thin community `Community 30`** (2 nodes): `RecommendableVideoUnitReader`, `UnitInventoryReader`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 31`** (2 nodes): `SemanticSpanReader`, `TranscriptSentenceReader`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 32`** (2 nodes): `Recommendation Transaction Manager`, `WithinTx Flow`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 33`** (2 nodes): `CandidateGenerator`, `candidate_generator.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 34`** (2 nodes): `Recommendation Selection Logic`, `Video Selector Interface`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 35`** (2 nodes): `evidence_resolver.go`, `EvidenceResolver`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 36`** (2 nodes): `querier.go`, `Querier`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 37`** (2 nodes): `Embedded Postgres Test Deviation`, `Real Postgres Test Layout`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 38`** (1 nodes): `DATABASE_URL Loader`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 39`** (1 nodes): `doc.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 40`** (1 nodes): `Explanation Builder Unit Test Suite`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 41`** (1 nodes): `Demand Planner Unit Test Suite`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 42`** (1 nodes): `Video Evidence Aggregator Unit Test Suite`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 43`** (1 nodes): `VideoUserStateReader`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 44`** (1 nodes): `RecommendationAuditRepository`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 45`** (1 nodes): `UnitServingStateRepository`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 46`** (1 nodes): `VideoServingStateRepository`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 47`** (1 nodes): `Generate Video Recommendations Usecase`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 48`** (1 nodes): `doc.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 49`** (1 nodes): `doc.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 50`** (1 nodes): `doc.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 51`** (1 nodes): `Context Assembler Interface`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 52`** (1 nodes): `doc.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 53`** (1 nodes): `doc.go`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 54`** (1 nodes): `Evidence Resolution Scope`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 55`** (1 nodes): `Learning-to-Recommendation Replay Consistency E2E`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 56`** (1 nodes): `Shared E2E Harness Bootstrap`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 57`** (1 nodes): `Content Facts and Recall-ready Indexes`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 58`** (1 nodes): `TargetUnitSpec`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 59`** (1 nodes): `RecordLearningEventsRequest`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 60`** (1 nodes): `ReplayUserStatesRequest`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 61`** (1 nodes): `TargetStateCommandRepository Port`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 62`** (1 nodes): `UnitLearningEventRepository Port`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 63`** (1 nodes): `ErrLateStrongEvent`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 64`** (1 nodes): `List User Unit States Usecase`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 65`** (1 nodes): `Ensure Target Units Usecase`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 66`** (1 nodes): `Set Target Inactive Usecase`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 67`** (1 nodes): `Resume Target Unit Usecase`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 68`** (1 nodes): `Record Learning Events Usecase`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 69`** (1 nodes): `Replay User States Usecase`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 70`** (1 nodes): `Reducer and Aggregate Root Package Scope`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 71`** (1 nodes): `User Unit State Filter`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 72`** (1 nodes): `Target Unit Spec`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 73`** (1 nodes): `生成统一审计上下文。          审计上下文尽量只放排障需要的信息，不放整坨 transcript。`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 74`** (1 nodes): `__init__.py`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 75`** (1 nodes): `表示 normalizer 阶段产出的基础行集合。      这里故意不包含 transcript 摘要和 unit index。     原因是这两类数据属于`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 76`** (1 nodes): `表示 normalizer 和 index_builder 产出的完整写库数据。`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 77`** (1 nodes): `表示写入审计表时需要的字段集合。      这里不包含 ingestion_record_id 和时间戳，因为这两个值应由 repository 在真正写库时生`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 78`** (1 nodes): `表示数据库里已存在的 clip 快照。      这个对象专门给 main 做“是否可以 skipped”判断用。     它只保留幂等判断所需的字段，不承担完`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 79`** (1 nodes): `表示 main 汇总时使用的单 clip 最终结果。`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 80`** (1 nodes): `Single Reducer Rule`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 81`** (1 nodes): `RecordLearningEvents Pipeline`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 82`** (1 nodes): `ReplayUserStates Pipeline`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 83`** (1 nodes): `Short Final-Write Transactions`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 84`** (1 nodes): `Run/Item Audit Center Policy`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 85`** (1 nodes): `Read-Only Upstream Data Policy`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 86`** (1 nodes): `video_recommendation_runs / video_recommendation_items`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 87`** (1 nodes): `user_unit_serving_states`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 88`** (1 nodes): `user_video_serving_states`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.
- **Thin community `Community 89`** (1 nodes): `Cross-Module E2E Test Scope`
  Too small to be a meaningful cluster - may be noise or needs more connections extracted.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Reduce()` connect `Community 15` to `Community 1`?**
  _High betweenness centrality (0.024) - this node is a cross-community bridge._
- **Why does `_load_from_parent_file()` connect `Community 0` to `Community 4`?**
  _High betweenness centrality (0.022) - this node is a cross-community bridge._
- **Are the 49 inferred relationships involving `CatalogIngestError` (e.g. with `ValidationWarning` and `表示校验阶段发现的非阻断性告警。      这类问题不会阻止当前 clip 入库，但需要：     - 在命令行结果里暴露     - 在审计表 warning`) actually correct?**
  _`CatalogIngestError` has 49 INFERRED edges - model-reasoned connections that need verification._
- **Are the 20 inferred relationships involving `Harness` (e.g. with `TestE2E_RecommendationWritesAuditAndServingStateWithEvidence()` and `TestE2E_RecommendationSecondRunAppliesServingAndWatchedPenalty()`) actually correct?**
  _`Harness` has 20 INFERRED edges - model-reasoned connections that need verification._
- **Are the 29 inferred relationships involving `LoadedClipInput` (e.g. with `ValidationWarning` and `表示校验阶段发现的非阻断性告警。      这类问题不会阻止当前 clip 入库，但需要：     - 在命令行结果里暴露     - 在审计表 warning`) actually correct?**
  _`LoadedClipInput` has 29 INFERRED edges - model-reasoned connections that need verification._
- **Are the 10 inferred relationships involving `CatalogRepository` (e.g. with `脚本主入口。      main 只负责总编排：     - 读取参数     - 初始化 repository     - 调用 loader / valid` and `判断当前 clip 是否可直接 skipped。      这里严格按 README 中的“无变化跳过”规则比较。     只要 transcript chec`) actually correct?**
  _`CatalogRepository` has 10 INFERRED edges - model-reasoned connections that need verification._
- **Are the 22 inferred relationships involving `TestE2E_RecommendationSelectorRespectsFallbackMaxAndCoreDominantMin()` (e.g. with `Harness` and `.LearningSuite()`) actually correct?**
  _`TestE2E_RecommendationSelectorRespectsFallbackMaxAndCoreDominantMin()` has 22 INFERRED edges - model-reasoned connections that need verification._