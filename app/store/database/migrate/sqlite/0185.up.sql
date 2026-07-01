with
repositories_calcs(repo_calc_id, repo_calc_num_open_pulls, repo_calc_num_closed_pulls, repo_calc_num_merged_pulls, repo_calc_num_pulls) as (
    select
		repo_id,
		count(*) filter (where pullreq_state = 'open'),
		count(*) filter (where pullreq_state = 'closed'),
		count(*) filter (where pullreq_state = 'merged'),
		count(pullreq_target_repo_id)
    from repositories
    left join pullreqs on pullreq_target_repo_id = repo_id
    group by repo_id
),
repositories_mismatch(repo_calc_id, repo_calc_num_open_pulls, repo_calc_num_closed_pulls, repo_calc_num_merged_pulls, repo_calc_num_pulls) as (
	select repo_calc_id, repo_calc_num_open_pulls, repo_calc_num_closed_pulls, repo_calc_num_merged_pulls, repo_calc_num_pulls
	from repositories
	inner join repositories_calcs on repo_calc_id = repo_id
    where
		repo_num_open_pulls != repo_calc_num_open_pulls
		or repo_num_closed_pulls != repo_calc_num_closed_pulls
		or repo_num_merged_pulls != repo_calc_num_merged_pulls
		or repo_num_pulls != repo_calc_num_pulls
)
update repositories
set
	repo_num_open_pulls = repo_calc_num_open_pulls,
	repo_num_closed_pulls = repo_calc_num_closed_pulls,
	repo_num_merged_pulls = repo_calc_num_merged_pulls,
	repo_num_pulls = repo_calc_num_pulls
from repositories_mismatch
where repo_calc_id = repo_id;
